package sap_api_caller

import (
	"fmt"
	"io/ioutil"
	"net/http"
	sap_api_output_formatter "sap-api-integrations-employee-basic-data-reads-rmq-kube/SAP_API_Output_Formatter"
	"strings"
	"sync"

	"github.com/latonaio/golang-logging-library-for-sap/logger"
	"golang.org/x/xerrors"
)

type RMQOutputter interface {
	Send(sendQueue string, payload map[string]interface{}) error
}

type SAPAPICaller struct {
	baseURL      string
	apiKey       string
	outputQueues []string
	outputter    RMQOutputter
	log          *logger.Logger
}

func NewSAPAPICaller(baseUrl string, outputQueueTo []string, outputter RMQOutputter, l *logger.Logger) *SAPAPICaller {
	return &SAPAPICaller{
		baseURL:      baseUrl,
		apiKey:       GetApiKey(),
		outputQueues: outputQueueTo,
		outputter:    outputter,
		log:          l,
	}
}

func (c *SAPAPICaller) AsyncGetEmployeeBasicData(employeeID, userID string, accepter []string) {
	wg := &sync.WaitGroup{}
	wg.Add(len(accepter))
	for _, fn := range accepter {
		switch fn {
		case "BusinessUserCollection":
			func() {
				c.BusinessUserCollection(employeeID)
				wg.Done()
			}()
		case "EmployeeBasicData":
			func() {
				c.EmployeeBasicData(userID)
				wg.Done()
			}()
		default:
			wg.Done()
		}
	}

	wg.Wait()
}

func (c *SAPAPICaller) BusinessUserCollection(employeeID string) {
	businessUserCollectionData, err := c.callEmployeeBasicDataSrvAPIRequirementBusinessUserCollection("BusinessUserCollectionData", employeeID)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": businessUserCollectionData, "function": "BusinessUserCollectionData"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(businessUserCollectionData)

	businessUserBusinessRoleAssignmentData, err := c.callToBusinessUserBusinessRoleAssignment(businessUserCollectionData[0].ToBusinessUserBusinessRoleAssignment)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": businessUserBusinessRoleAssignmentData, "function": "BusinessUserBusinessRoleAssignmentData"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(businessUserBusinessRoleAssignmentData)

}

func (c *SAPAPICaller) callEmployeeBasicDataSrvAPIRequirementBusinessUserCollection(api, employeeID string) ([]sap_api_output_formatter.BusinessUserCollection, error) {
	url := strings.Join([]string{c.baseURL, "c4codataapi", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithBusinessUserCollection(req, employeeID)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToBusinessUserCollection(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) callToBusinessUserBusinessRoleAssignment(url string) ([]sap_api_output_formatter.ToBusinessUserBusinessRoleAssignment, error) {
	req, _ := http.NewRequest("GET", url, nil)
	c.setHeaderAPIKeyAccept(req)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToToBusinessUserBusinessRoleAssignment(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) EmployeeBasicData(userID string) {
	employeeBasicDataData, err := c.callEmployeeBasicDataSrvAPIRequirementEmployeeBasicData("EmployeeBasicDataData", userID)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": employeeBasicDataData, "function": "EmployeeBasicDataData"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(employeeBasicDataData)
}

func (c *SAPAPICaller) callEmployeeBasicDataSrvAPIRequirementEmployeeBasicData(api, userID string) ([]sap_api_output_formatter.EmployeeBasicData, error) {
	url := strings.Join([]string{c.baseURL, "c4codataapi", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithEmployeeBasicData(req, userID)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToEmployeeBasicData(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) setHeaderAPIKeyAccept(req *http.Request) {
	req.Header.Set("APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
}

func (c *SAPAPICaller) getQueryWithBusinessUserCollection(req *http.Request, employeeID string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("EmployeeID eq '%s'", employeeID))
	req.URL.RawQuery = params.Encode()
}

func (c *SAPAPICaller) getQueryWithEmployeeBasicData(req *http.Request, userID string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("UserID eq '%s'", userID))
	req.URL.RawQuery = params.Encode()
}