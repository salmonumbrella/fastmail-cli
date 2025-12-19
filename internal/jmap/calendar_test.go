package jmap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetCalendars(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		statusCode     int
		wantErr        bool
		wantCalendars  int
		checkFirstName string
	}{
		{
			name:       "successful retrieval",
			statusCode: http.StatusOK,
			responseBody: `{
				"methodResponses": [
					[
						"Calendar/get",
						{
							"accountId": "test-account",
							"state": "state-123",
							"list": [
								{
									"id": "cal1",
									"name": "Personal",
									"color": "#FF0000",
									"isVisible": true,
									"isSubscribed": false
								},
								{
									"id": "cal2",
									"name": "Work",
									"color": "#0000FF",
									"isVisible": true,
									"isSubscribed": false
								}
							]
						},
						"c0"
					]
				]
			}`,
			wantErr:        false,
			wantCalendars:  2,
			checkFirstName: "Personal",
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			responseBody: `{
				"methodResponses": [
					[
						"Calendar/get",
						{
							"accountId": "test-account",
							"state": "state-123",
							"list": []
						},
						"c0"
					]
				]
			}`,
			wantErr:       false,
			wantCalendars: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create API server
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer apiServer.Close()

			// Create session server
			sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"apiUrl": "` + apiServer.URL + `",
					"downloadUrl": "` + apiServer.URL + `",
					"accounts": {"acc123": {}},
					"primaryAccounts": {
						"urn:ietf:params:jmap:mail": "acc123",
						"urn:ietf:params:jmap:calendars": "acc123"
					},
					"capabilities": {
						"urn:ietf:params:jmap:core": {},
						"urn:ietf:params:jmap:calendars": {}
					}
				}`))
			}))
			defer sessionServer.Close()

			client := NewClient("test-token")
			client.baseURL = sessionServer.URL

			calendars, err := client.GetCalendars(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCalendars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(calendars) != tt.wantCalendars {
				t.Errorf("GetCalendars() got %d calendars, want %d", len(calendars), tt.wantCalendars)
			}

			if tt.checkFirstName != "" && len(calendars) > 0 {
				if calendars[0].Name != tt.checkFirstName {
					t.Errorf("GetCalendars() first calendar name = %v, want %v", calendars[0].Name, tt.checkFirstName)
				}
			}
		})
	}
}

func TestCalendarsNotEnabled(t *testing.T) {
	// Create session server without calendars capability
	sessionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"apiUrl": "http://example.com",
			"downloadUrl": "http://example.com",
			"accounts": {"acc123": {}},
			"primaryAccounts": {
				"urn:ietf:params:jmap:mail": "acc123"
			},
			"capabilities": {
				"urn:ietf:params:jmap:core": {}
			}
		}`))
	}))
	defer sessionServer.Close()

	client := NewClient("test-token")
	client.baseURL = sessionServer.URL

	_, err := client.GetCalendars(context.Background())
	if err != ErrCalendarsNotEnabled {
		t.Errorf("GetCalendars() error = %v, want %v", err, ErrCalendarsNotEnabled)
	}

	_, err = client.GetEvents(context.Background(), "cal1", time.Time{}, time.Time{}, 10)
	if err != ErrCalendarsNotEnabled {
		t.Errorf("GetEvents() error = %v, want %v", err, ErrCalendarsNotEnabled)
	}

	_, err = client.GetEventByID(context.Background(), "event1")
	if err != ErrCalendarsNotEnabled {
		t.Errorf("GetEventByID() error = %v, want %v", err, ErrCalendarsNotEnabled)
	}

	_, err = client.CreateEvent(context.Background(), &CalendarEvent{})
	if err != ErrCalendarsNotEnabled {
		t.Errorf("CreateEvent() error = %v, want %v", err, ErrCalendarsNotEnabled)
	}

	_, err = client.UpdateEvent(context.Background(), "event1", map[string]interface{}{})
	if err != ErrCalendarsNotEnabled {
		t.Errorf("UpdateEvent() error = %v, want %v", err, ErrCalendarsNotEnabled)
	}

	err = client.DeleteEvent(context.Background(), "event1")
	if err != ErrCalendarsNotEnabled {
		t.Errorf("DeleteEvent() error = %v, want %v", err, ErrCalendarsNotEnabled)
	}
}
