// Copyright 2018 JDCLOUD.COM
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
//
// NOTE: This class is auto generated by the jdcloud code generator program.

package apis

import (
    "github.com/jdcloud-api/jdcloud-sdk-go/core"
    vm "github.com/jdcloud-api/jdcloud-sdk-go/services/vm/models"
)

type CreateImageRequest struct {

    core.JDCloudRequest

    /* 地域ID  */
    RegionId string `json:"regionId"`

    /* 云主机ID  */
    InstanceId string `json:"instanceId"`

    /* 镜像名称，<a href="http://docs.jdcloud.com/virtual-machines/api/general_parameters">参考公共参数规范</a>。  */
    Name string `json:"name"`

    /* 镜像描述，<a href="http://docs.jdcloud.com/virtual-machines/api/general_parameters">参考公共参数规范</a>。 (Optional) */
    Description *string `json:"description"`

    /* 数据盘列表，可以在实例已挂载数据盘的基础上，额外增加新的快照、空盘、或排除云主机中的数据盘。 (Optional) */
    DataDisks []vm.InstanceDiskAttachmentSpec `json:"dataDisks"`
}

/*
 * param regionId: 地域ID (Required)
 * param instanceId: 云主机ID (Required)
 * param name: 镜像名称，<a href="http://docs.jdcloud.com/virtual-machines/api/general_parameters">参考公共参数规范</a>。 (Required)
 *
 * @Deprecated, not compatible when mandatory parameters changed
 */
func NewCreateImageRequest(
    regionId string,
    instanceId string,
    name string,
) *CreateImageRequest {

	return &CreateImageRequest{
        JDCloudRequest: core.JDCloudRequest{
			URL:     "/regions/{regionId}/instances/{instanceId}:createImage",
			Method:  "POST",
			Header:  nil,
			Version: "v1",
		},
        RegionId: regionId,
        InstanceId: instanceId,
        Name: name,
	}
}

/*
 * param regionId: 地域ID (Required)
 * param instanceId: 云主机ID (Required)
 * param name: 镜像名称，<a href="http://docs.jdcloud.com/virtual-machines/api/general_parameters">参考公共参数规范</a>。 (Required)
 * param description: 镜像描述，<a href="http://docs.jdcloud.com/virtual-machines/api/general_parameters">参考公共参数规范</a>。 (Optional)
 * param dataDisks: 数据盘列表，可以在实例已挂载数据盘的基础上，额外增加新的快照、空盘、或排除云主机中的数据盘。 (Optional)
 */
func NewCreateImageRequestWithAllParams(
    regionId string,
    instanceId string,
    name string,
    description *string,
    dataDisks []vm.InstanceDiskAttachmentSpec,
) *CreateImageRequest {

    return &CreateImageRequest{
        JDCloudRequest: core.JDCloudRequest{
            URL:     "/regions/{regionId}/instances/{instanceId}:createImage",
            Method:  "POST",
            Header:  nil,
            Version: "v1",
        },
        RegionId: regionId,
        InstanceId: instanceId,
        Name: name,
        Description: description,
        DataDisks: dataDisks,
    }
}

/* This constructor has better compatible ability when API parameters changed */
func NewCreateImageRequestWithoutParam() *CreateImageRequest {

    return &CreateImageRequest{
            JDCloudRequest: core.JDCloudRequest{
            URL:     "/regions/{regionId}/instances/{instanceId}:createImage",
            Method:  "POST",
            Header:  nil,
            Version: "v1",
        },
    }
}

/* param regionId: 地域ID(Required) */
func (r *CreateImageRequest) SetRegionId(regionId string) {
    r.RegionId = regionId
}

/* param instanceId: 云主机ID(Required) */
func (r *CreateImageRequest) SetInstanceId(instanceId string) {
    r.InstanceId = instanceId
}

/* param name: 镜像名称，<a href="http://docs.jdcloud.com/virtual-machines/api/general_parameters">参考公共参数规范</a>。(Required) */
func (r *CreateImageRequest) SetName(name string) {
    r.Name = name
}

/* param description: 镜像描述，<a href="http://docs.jdcloud.com/virtual-machines/api/general_parameters">参考公共参数规范</a>。(Optional) */
func (r *CreateImageRequest) SetDescription(description string) {
    r.Description = &description
}

/* param dataDisks: 数据盘列表，可以在实例已挂载数据盘的基础上，额外增加新的快照、空盘、或排除云主机中的数据盘。(Optional) */
func (r *CreateImageRequest) SetDataDisks(dataDisks []vm.InstanceDiskAttachmentSpec) {
    r.DataDisks = dataDisks
}

// GetRegionId returns path parameter 'regionId' if exist,
// otherwise return empty string
func (r CreateImageRequest) GetRegionId() string {
    return r.RegionId
}

type CreateImageResponse struct {
    RequestID string `json:"requestId"`
    Error core.ErrorResponse `json:"error"`
    Result CreateImageResult `json:"result"`
}

type CreateImageResult struct {
    ImageId string `json:"imageId"`
}