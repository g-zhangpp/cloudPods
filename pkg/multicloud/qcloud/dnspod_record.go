// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package qcloud

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRecordCreateRet struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Weight int    `json:"weight"`
}

type SRecordCountInfo struct {
	RecordTotal string `json:"record_total"`
	RecordsNum  string `json:"records_num"`
	SubDomains  string `json:"sub_domains"`
}

type SDnsRecord struct {
	domain     *SDomian
	ID         int    `json:"id"`
	TTL        int    `json:"ttl"`
	Value      string `json:"value"`
	Enabled    int    `json:"enabled"`
	Status     string `json:"status"`
	UpdatedOn  string `json:"updated_on"`
	QProjectID int    `json:"q_project_id"`
	Name       string `json:"name"`
	Line       string `json:"line"`
	LineID     string `json:"line_id"`
	Type       string `json:"type"`
	Remark     string `json:"remark"`
	Mx         int64  `json:"mx"`
	Hold       string `json:"hold"`
}

// https://cloud.tencent.com/document/product/302/8517
func (client *SQcloudClient) GetDnsRecords(projectId string, sDomainName string, offset int, limit int) ([]SDnsRecord, int, error) {

	params := map[string]string{}
	params["offset"] = strconv.Itoa(offset)
	params["length"] = strconv.Itoa(limit)
	params["domain"] = sDomainName
	if len(projectId) > 0 {
		params["qProjectId"] = projectId
	}
	resp, err := client.cnsRequest("RecordList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "client.cnsRequest(RecordList, %s)", jsonutils.Marshal(params).String())
	}
	count := SRecordCountInfo{}
	err = resp.Unmarshal(&count, "info")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "%s.Unmarshal(info)", jsonutils.Marshal(resp).String())
	}
	records := []SDnsRecord{}
	err = resp.Unmarshal(&records, "records")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "%s.Unmarshal(records)", jsonutils.Marshal(resp).String())
	}
	RecordTotal, err := strconv.Atoi(count.RecordTotal)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "strconv.Atoi(%s)", count.RecordTotal)
	}
	return records, RecordTotal, nil
}

func (client *SQcloudClient) GetAllDnsRecords(sDomainName string) ([]SDnsRecord, error) {
	count := 0
	result := []SDnsRecord{}
	for true {
		// -1 ????????????; 0,default????????????
		records, total, err := client.GetDnsRecords("-1", sDomainName, count, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "client.GetDnsRecords(%s,%d,%d)", sDomainName, count, 100)
		}

		result = append(result, records...)
		count += len(records)
		if total <= count {
			break
		}
	}
	return result, nil
}

func GetRecordLineLineType(policyinfo cloudprovider.TDnsPolicyValue) string {
	switch policyinfo {
	case cloudprovider.DnsPolicyValueMainland:
		return "??????"
	case cloudprovider.DnsPolicyValueOversea:
		return "??????"
	case cloudprovider.DnsPolicyValueTelecom:
		return "??????"
	case cloudprovider.DnsPolicyValueUnicom:
		return "??????"
	case cloudprovider.DnsPolicyValueChinaMobile:
		return "??????"
	case cloudprovider.DnsPolicyValueCernet:
		return "?????????"

	case cloudprovider.DnsPolicyValueBaidu:
		return "??????"
	case cloudprovider.DnsPolicyValueGoogle:
		return "??????"
	case cloudprovider.DnsPolicyValueYoudao:
		return "??????"
	case cloudprovider.DnsPolicyValueBing:
		return "??????"
	case cloudprovider.DnsPolicyValueSousou:
		return "??????"
	case cloudprovider.DnsPolicyValueSougou:
		return "??????"
	case cloudprovider.DnsPolicyValueQihu360:
		return "??????"
	default:
		return "??????"
	}
}

// https://cloud.tencent.com/document/api/302/8516
func (client *SQcloudClient) CreateDnsRecord(opts *cloudprovider.DnsRecordSet, domainName string) (string, error) {
	params := map[string]string{}
	recordline := GetRecordLineLineType(opts.PolicyValue)
	if opts.Ttl < 600 {
		opts.Ttl = 600
	}
	if opts.Ttl > 604800 {
		opts.Ttl = 604800
	}
	if len(opts.DnsName) < 1 {
		opts.DnsName = "@"
	}
	params["domain"] = domainName
	params["subDomain"] = opts.DnsName
	params["recordType"] = string(opts.DnsType)
	params["ttl"] = strconv.FormatInt(opts.Ttl, 10)
	params["value"] = opts.DnsValue
	params["recordLine"] = recordline
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["mx"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	resp, err := client.cnsRequest("RecordCreate", params)
	if err != nil {
		return "", errors.Wrapf(err, "client.cnsRequest(RecordCreate, %s)", jsonutils.Marshal(params).String())
	}
	SRecordCreateRet := SRecordCreateRet{}
	err = resp.Unmarshal(&SRecordCreateRet, "record")
	if err != nil {
		return "", errors.Wrapf(err, "%s.Unmarshal(records)", jsonutils.Marshal(resp).String())
	}
	return SRecordCreateRet.ID, nil
}

// https://cloud.tencent.com/document/product/302/8511
func (client *SQcloudClient) ModifyDnsRecord(opts *cloudprovider.DnsRecordSet, domainName string) error {
	params := map[string]string{}
	recordline := GetRecordLineLineType(opts.PolicyValue)
	if opts.Ttl < 600 {
		opts.Ttl = 600
	}
	if opts.Ttl > 604800 {
		opts.Ttl = 604800
	}
	subDomain := strings.TrimSuffix(opts.DnsName, "."+domainName)
	if len(subDomain) < 1 {
		subDomain = "@"
	}
	params["domain"] = domainName
	params["recordId"] = opts.ExternalId
	params["subDomain"] = subDomain
	params["recordType"] = string(opts.DnsType)
	params["ttl"] = strconv.FormatInt(opts.Ttl, 10)
	params["value"] = opts.DnsValue
	params["recordLine"] = recordline
	if opts.DnsType == cloudprovider.DnsTypeMX {
		params["mx"] = strconv.FormatInt(opts.MxPriority, 10)
	}
	_, err := client.cnsRequest("RecordModify", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(RecordModify, %s)", jsonutils.Marshal(params).String())
	}
	return nil
}

// https://cloud.tencent.com/document/product/302/8519
func (client *SQcloudClient) ModifyRecordStatus(status, recordId, domain string) error {
	params := map[string]string{}
	params["domain"] = domain
	params["recordId"] = recordId
	params["status"] = status // ???disable??? ??? ???enable???
	_, err := client.cnsRequest("RecordStatus", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(RecordModify, %s)", jsonutils.Marshal(params).String())
	}
	return nil
}

// https://cloud.tencent.com/document/api/302/8514
func (client *SQcloudClient) DeleteDnsRecord(recordId int, domainName string) error {
	params := map[string]string{}
	params["domain"] = domainName
	params["recordId"] = strconv.Itoa(recordId)
	_, err := client.cnsRequest("RecordDelete", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(RecordDelete, %s)", jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SDnsRecord) GetGlobalId() string {
	return strconv.Itoa(self.ID)
}

func (self *SDnsRecord) GetDnsName() string {
	return self.Name
}

func (self *SDnsRecord) GetStatus() string {
	if self.Status != "spam" {
		return api.DNS_RECORDSET_STATUS_AVAILABLE
	}
	return api.DNS_ZONE_STATUS_UNKNOWN
}

func (self *SDnsRecord) GetEnabled() bool {
	return self.Enabled == 1
}

func (self *SDnsRecord) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SDnsRecord) GetDnsValue() string {
	if self.GetDnsType() == cloudprovider.DnsTypeMX || self.GetDnsType() == cloudprovider.DnsTypeCNAME || self.GetDnsType() == cloudprovider.DnsTypeSRV {
		return self.Value[:len(self.Value)-1]
	}
	return self.Value
}

func (self *SDnsRecord) GetTTL() int64 {
	return int64(self.TTL)
}

func (self *SDnsRecord) GetMxPriority() int64 {
	if self.GetDnsType() == cloudprovider.DnsTypeMX {
		return self.Mx
	}
	return 0
}

func (self *SDnsRecord) GetPolicyType() cloudprovider.TDnsPolicyType {
	switch self.Line {
	case "??????", "??????":
		return cloudprovider.DnsPolicyTypeByGeoLocation
	case "??????", "??????", "??????", "?????????":
		return cloudprovider.DnsPolicyTypeByCarrier
	case "??????", "??????", "??????", "??????", "??????", "??????", "??????":
		return cloudprovider.DnsPolicyTypeBySearchEngine
	default:
		return cloudprovider.DnsPolicyTypeSimple
	}
}

func (self *SDnsRecord) GetPolicyOptions() *jsonutils.JSONDict {
	return nil
}

func (self *SDnsRecord) GetPolicyValue() cloudprovider.TDnsPolicyValue {
	switch self.Line {
	case "??????":
		return cloudprovider.DnsPolicyValueMainland
	case "??????":
		return cloudprovider.DnsPolicyValueOversea

	case "??????":
		return cloudprovider.DnsPolicyValueTelecom
	case "??????":
		return cloudprovider.DnsPolicyValueUnicom
	case "??????":
		return cloudprovider.DnsPolicyValueChinaMobile
	case "?????????":
		return cloudprovider.DnsPolicyValueCernet

	case "??????":
		return cloudprovider.DnsPolicyValueBaidu
	case "??????":
		return cloudprovider.DnsPolicyValueGoogle
	case "??????":
		return cloudprovider.DnsPolicyValueYoudao
	case "??????":
		return cloudprovider.DnsPolicyValueBing
	case "??????":
		return cloudprovider.DnsPolicyValueSousou
	case "??????":
		return cloudprovider.DnsPolicyValueSougou
	case "??????":
		return cloudprovider.DnsPolicyValueQihu360
	default:
		return cloudprovider.DnsPolicyValueEmpty
	}
}
