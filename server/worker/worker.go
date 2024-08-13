package worker

import (
	"encoding/json"
	"github.com/go-kit/kit/log/level"
)

type Dashboard struct {
	Uid   string
	Data  map[string]interface{}
	Title string
}

type DashboardsResponse struct {
	Uid string `json:"uid"`
}

type DashboardResponse struct {
	Dashboard map[string]interface{} `json:"dashboard"`
}

type DashboardUpdateResponse struct {
	FolderUid string `json:"folderUid"`
	Id        uint64 `json:"id"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	Uid       string `json:"uid"`
	Url       string `json:"url"`
	Version   uint64 `json:"version"`
}

func getTypedDashboardData(rawDashboardData *Dashboard) map[string]interface{} {
	dashboard, ok := rawDashboardData.Data["dashboard"]
	var emptyDashboardData map[string]interface{}
	if !ok {
		return emptyDashboardData
	}

	dashboardData, ok := dashboard.(map[string]interface{})
	if !ok {
		return emptyDashboardData
	}

	return dashboardData
}

func getTypedPanelsData(dashboardData map[string]interface{}) []interface{} {
	var emptyResponse []interface{}

	// Dashboards data might contain folders or other nonusable "dashboards"
	// All usable dashboards have minimum of empty panels array, even when dashboard is newly created without any panels
	// Due to this, we do not handle non "ok"
	panels, ok := dashboardData["panels"]
	if !ok {
		return emptyResponse
	}

	panelList, ok := panels.([]interface{})
	if !ok {
		return emptyResponse
	}

	return panelList
}

func isAnalyticsPanel(panel map[string]interface{}) bool {
	panelType, ok := panel["type"]
	if ok {
		return panelType == "macropower-analytics-panel"
	}

	return false
}

func (api *Client) GetDashboard(uid string) *Dashboard {
	res, err := api.Get("/api/dashboards/uid/" + uid)
	if err != nil {
		level.Info(api.Logger).Log(
			"status", "error",
			"message", "GetDashboard - api.Get failed",
			"error", err,
		)

		return nil
	}

	var dashboardData map[string]interface{}
	err = json.Unmarshal(res, &dashboardData)
	if err != nil {
		level.Info(api.Logger).Log(
			"status", "error",
			"message", "GetDashboard - Failed to parse dashboardData response",
			"error", err,
		)

		return nil
	}

	return &Dashboard{
		Uid:  uid,
		Data: dashboardData,
	}
}

func (api *Client) updateDashboards(dashboards []Dashboard) {
	for _, dashboard := range dashboards {
		if dashboard.Title == "test dashboard - empty" {
			dashboard.Data["overwrite"] = true
			dashboard.Data["message"] = "PRO - Auto-add analytics panel"

			payload, err := json.Marshal(dashboard.Data)
			if err != nil {
				return
			}

			res, err := api.Post("/api/dashboards/db", payload)
			if err != nil {
				level.Info(api.Logger).Log(
					"status", "error",
					"message", "updateDashboards - Failed to update dashboard",
					"error", err,
				)

				return
			}

			var responseAsStruct DashboardUpdateResponse
			err = json.Unmarshal(res, &responseAsStruct)
			if err != nil {
				level.Info(api.Logger).Log(
					"status", "error",
					"message", "updateDashboards - Failed to parse update response",
					"error", err,
				)

				return
			}

			if responseAsStruct.Status == "success" {
				level.Info(api.Logger).Log(
					"status", "success",
					"message", "Added analytics to "+dashboard.Title,
					"error", err,
				)
			}
		}
	}
}

func (api *Client) getAnalyticsPanelData(panelId int) map[string]interface{} {
	return map[string]interface{}{
		"id":    panelId,
		"title": "Analytics",
		"type":  "macropower-analytics-panel",
		"gridPos": map[string]interface{}{
			"h": 0,
			"w": 0,
			"x": 0,
			"y": 0,
		},
		"options": map[string]interface{}{
			"analyticsOptions": map[string]interface{}{
				"dashboard":         "$__dashboard",
				"flatten":           false,
				"heartbeatAlways":   false,
				"heartbeatInterval": 60,
				"postEnd":           false,
				"postHeartbeat":     false,
				"postStart":         true,
				"server":            api.AnalyticsUrl + "/write",
			},
		},
	}
}

func (api *Client) AddAnalyticsToDashboards() {
	res, err := api.Get("/api/search")
	if err != nil {
		level.Info(api.Logger).Log(
			"status", "error",
			"message", "AddAnalyticsToDashboards - Failed to get dashboard data - /api/search",
			"error", err,
		)

		return
	}

	var response []DashboardsResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		level.Info(api.Logger).Log(
			"status", "error",
			"message", "AddAnalyticsToDashboards - Failed to parse JSON response",
			"error", err,
		)

		return
	}

	var dashboardsToUpdate []Dashboard
	for _, dashboardEntry := range response {
		hasAnalyticsPanel := false
		rawDashboardData := api.GetDashboard(dashboardEntry.Uid)
		dashboardData := getTypedDashboardData(rawDashboardData)
		panels := getTypedPanelsData(dashboardData)

		largestPanelId := 0
		for _, panel := range panels {
			panelMap, ok := panel.(map[string]interface{})
			if !ok {
				continue
			}

			// Go reports id as float, instead of int
			panelId := panelMap["id"].(float64)
			panelIdAsInt := int(panelId)
			if largestPanelId < panelIdAsInt {
				largestPanelId = panelIdAsInt
			}

			if isAnalyticsPanel(panelMap) {
				hasAnalyticsPanel = true

				break
			}
		}

		title, ok := dashboardData["title"]
		if !hasAnalyticsPanel && ok {
			newAnalyticsPanel := api.getAnalyticsPanelData(largestPanelId + 1)
			panels = append([]interface{}{newAnalyticsPanel}, panels...)
			dashboardData["panels"] = panels
			rawDashboardData.Data["dashboard"] = dashboardData

			dashboardsToUpdate = append(dashboardsToUpdate, Dashboard{
				Uid:   dashboardEntry.Uid,
				Data:  rawDashboardData.Data,
				Title: title.(string),
			})
		}
	}

	api.updateDashboards(dashboardsToUpdate)
}
