/*
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/apache/incubator-yunikorn-k8shim/pkg/apis/yunikorn.apache.org/v1alpha1"
	"github.com/apache/incubator-yunikorn-k8shim/pkg/common/constants"
)

func FindAppTaskGroup(appTaskGroups []*v1alpha1.TaskGroup, groupName string) (*v1alpha1.TaskGroup, error) {
	if groupName == "" {
		// task has no group defined
		return nil, nil
	}

	// app has no taskGroups associated
	if len(appTaskGroups) == 0 {
		return nil, nil
	}

	// task group defined in app, return the corresponding taskGroup
	for _, tg := range appTaskGroups {
		if tg.Name == groupName {
			return tg, nil
		}
	}

	// task group name specified, but could not find a mapping value in app taskGroups
	return nil, fmt.Errorf("taskGroup %s is not defined in the application", groupName)
}

// the placeholder name is the pod name, pod name can not be longer than 63 chars,
// taskGroup name and appID will be truncated if they go over 20/28 chars respectively,
// each taskGroup is assigned with an incremental index starting from 0.
func GeneratePlaceholderName(taskGroupName, appID string, index int32) string {
	// taskGroup name no longer than 20 chars
	// appID no longer than 28 chars
	// total length no longer than 20 + 28 + 5 + 10 = 63
	shortTaskGroupName := fmt.Sprintf("%.20s", taskGroupName)
	shortAppID := fmt.Sprintf("%.28s", appID)
	return "tg-" + shortTaskGroupName + "-" + shortAppID + fmt.Sprintf("-%d", index)
}

func GetPlaceholderResourceRequest(resources map[string]resource.Quantity) v1.ResourceList {
	resourceReq := v1.ResourceList{}
	for k, v := range resources {
		resourceReq[v1.ResourceName(k)] = v
	}
	return resourceReq
}

func GetPlaceholderFlagFromPodSpec(pod *v1.Pod) bool {
	if value, ok := pod.Annotations[constants.AnnotationPlaceholderFlag]; ok {
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}

	if value, ok := pod.Labels[constants.LabelPlaceholderFlag]; ok {
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return false
}

func GetTaskGroupFromPodSpec(pod *v1.Pod) string {
	if value, ok := pod.Annotations[constants.AnnotationTaskGroupName]; ok {
		return value
	}
	return ""
}

func GetTaskGroupsFromAnnotation(pod *v1.Pod) ([]v1alpha1.TaskGroup, error) {
	taskGroupInfo, ok := pod.Annotations[constants.AnnotationTaskGroups]
	if !ok {
		return nil, nil
	}
	taskGroups := []v1alpha1.TaskGroup{}
	err := json.Unmarshal([]byte(taskGroupInfo), &taskGroups)
	if err != nil {
		return nil, err
	}
	// json.Unmarchal won't return error if name or MinMember is empty, but will return error if MinResource is empty or error format.
	for _, taskGroup := range taskGroups {
		if taskGroup.Name == "" {
			return nil, fmt.Errorf("can't get taskGroup Name from pod annotation, %s",
				pod.Annotations[constants.AnnotationTaskGroups])
		}
		if taskGroup.MinMember == int32(0) {
			return nil, fmt.Errorf("can't get taskGroup MinMember from pod annotation, %s",
				pod.Annotations[constants.AnnotationTaskGroups])
		}
		if taskGroup.MinMember < int32(0) {
			return nil, fmt.Errorf("minMember cannot be negative, %s",
				pod.Annotations[constants.AnnotationTaskGroups])
		}
	}
	return taskGroups, nil
}

func GetPlaceholderTimeoutParam(pod *v1.Pod) (int64, error) {
	param, ok := pod.Annotations[constants.AnnotationSchedulingPolicyParam]
	if !ok {
		return 0, nil
	}
	params := strings.Split(param, constants.SchedulingPolicyParamDelimiter)
	for _, p := range params {
		timeoutParam := strings.Split(p, "=")
		if timeoutParam[0] == constants.SchedulingPolicyTimeoutParam {
			if len(timeoutParam) != 2 {
				return 0, fmt.Errorf("unable to parse timeout value from annotation")
			}
			timeout, err := strconv.ParseInt(timeoutParam[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse timeout value: %s", timeoutParam[1])
			}
			return timeout, nil
		}
	}
	return 0, nil
}
