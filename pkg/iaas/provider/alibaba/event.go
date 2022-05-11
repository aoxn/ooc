package alibaba

import (
	"github.com/aoxn/wdrip/pkg/utils/log"
	rosc "github.com/denverdino/aliyungo/ros/standard"
	"time"
)

func ToResources(events []rosc.Event) []log.Resource {
	var resources []log.Resource
	for _, ev := range events {
		found := false
		for i := range resources {
			if resources[i].ResourceId == ev.LogicalResourceId {
				found = true
				// set if newer
				evtOut, err := time.ParseInLocation("2006-01-02T15:04:05", ev.CreateTime, time.UTC)
				if err != nil {
					continue
				}
				evtIn, err := time.ParseInLocation("2006-01-02T15:04:05", resources[i].UpdatedTime, time.UTC)
				if err != nil {
					continue
				}
				if evtOut.Before(evtIn) {
					resources[i].StartedTime = evtOut.Local().Format("2006-01-02T15:04:05")
				}
				if evtOut.After(evtIn) {
					resources[i].UpdatedTime = evtOut.Local().Format("2006-01-02T15:04:05")
					resources[i].EventId = ev.EventId
					resources[i].ResourceStatus = ev.Status
					resources[i].StatusReason = ev.StatusReason
					break
				}
			}
		}
		if !found {
			when, err := time.ParseInLocation("2006-01-02T15:04:05", ev.CreateTime, time.UTC)
			if err != nil {
				continue
			}
			resources = append(
				resources,
				log.Resource{
					StartedTime:    when.Local().Format("2006-01-02T15:04:05"),
					UpdatedTime:    when.Local().Format("2006-01-02T15:04:05"),
					EventId:        ev.EventId,
					ResourceType:   ev.ResourceType,
					ResourceId:     ev.LogicalResourceId,
					ResourceName:   ev.StackName,
					StatusReason:   ev.StatusReason,
					ResourceStatus: ev.Status,
				},
			)
		}
	}
	//fmt.Printf("%+v\n", resources)
	return resources
}
