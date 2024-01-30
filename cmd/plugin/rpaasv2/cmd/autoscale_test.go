// Copyright 2019 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"

	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/autogenerated"
)

func TestGetAutoscale(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args          []string
		expected      string
		expectedError string
		handler       http.Handler
	}{
		"when instance doesn't exist": {
			args: []string{"autoscale", "info", "-s", "my-service", "-i", "my-instance"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(autogenerated.Error{Msg: "instance \"my-instance\" not found"})
			}),
			expectedError: "could not get autoscale from RPaaS API: 404 Not Found",
		},

		"when autoscale is successfully returned": {
			args: []string{"autoscale", "info", "-s", "my-service", "-i", "my-instance"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(autogenerated.Autoscale{
					MinReplicas: 2,
					MaxReplicas: 5,
					Cpu:         autogenerated.PtrInt32(50),
					Memory:      autogenerated.PtrInt32(55),
					Rps:         autogenerated.PtrInt32(100),
				})
			}),
			expected: `min replicas: 2
max replicas: 5
+----------+-----------------+
| Triggers | trigger details |
+----------+-----------------+
| CPU      | 50%             |
| Memory   | 55%             |
| RPS      | 100 req/s       |
+----------+-----------------+
`,
		},

		"with schedules": {
			args: []string{"autoscale", "info", "-s", "my-service", "-i", "my-instance"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(autogenerated.Autoscale{
					MinReplicas: 0,
					MaxReplicas: 100,
					Schedules: []autogenerated.ScheduledWindow{
						{MinReplicas: 1, Start: "00 08 * * 1-5", End: "00 20 * * 1-5"},
						{MinReplicas: 5, Start: "00 20 * * 2", End: "00 01 * * 3"},
						{MinReplicas: 5, Start: "00 22 * * 0", End: "00 02 * * 1", Timezone: pointer.String("America/Chile")},
					},
				})
			}),
			expected: `min replicas: 0
max replicas: 100
+-------------+-------------------------------------------------------------+
|  Triggers   |                       trigger details                       |
+-------------+-------------------------------------------------------------+
| Schedule(s) | Window 1:                                                   |
|             |   Min replicas: 1                                           |
|             |   Start: At 08:00 AM, Monday through Friday (00 08 * * 1-5) |
|             |   End: At 08:00 PM, Monday through Friday (00 20 * * 1-5)   |
|             |                                                             |
|             | Window 2:                                                   |
|             |   Min replicas: 5                                           |
|             |   Start: At 08:00 PM, only on Tuesday (00 20 * * 2)         |
|             |   End: At 01:00 AM, only on Wednesday (00 01 * * 3)         |
|             |                                                             |
|             | Window 3:                                                   |
|             |   Min replicas: 5                                           |
|             |   Start: At 10:00 PM, only on Sunday (00 22 * * 0)          |
|             |   End: At 02:00 AM, only on Monday (00 02 * * 1)            |
|             |   Timezone: America/Chile                                   |
+-------------+-------------------------------------------------------------+
`,
		},

		"when get autoscale route is successful on JSON format": {
			args: []string{"autoscale", "info", "-s", "my-service", "-i", "my-instance", "--json"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(autogenerated.Autoscale{
					MinReplicas: 2,
					MaxReplicas: 5,
					Cpu:         autogenerated.PtrInt32(50),
					Memory:      autogenerated.PtrInt32(55),
					Rps:         autogenerated.PtrInt32(100),
				})
			}),
			expected: `{
	"cpu": 50,
	"maxReplicas": 5,
	"memory": 55,
	"minReplicas": 2,
	"rps": 100
}
`,
		},
	}

	for _, serverGen := range AllRpaasAPIServerGenerators {
		t.Run("", func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					require.NotNil(t, tt.handler, "you must provide an HTTP handler")
					server, args := serverGen(t, tt.handler)
					defer server.Close()

					args = append(args, tt.args...)

					var stdout bytes.Buffer
					err := NewApp(&stdout, io.Discard, nil).Run(args)
					if tt.expectedError != "" {
						assert.EqualError(t, err, tt.expectedError)
						return
					}

					require.NoError(t, err)
					assert.Equal(t, tt.expected, stdout.String())
				})
			}
		})
	}
}

func TestRemoveAutoscale(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args          []string
		expected      string
		expectedError string
		handler       http.HandlerFunc
	}{
		"when instance doesn't exist": {
			args: []string{"autoscale", "remove", "-s", "my-service", "-i", "my-instance"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(autogenerated.Error{Msg: "instance \"my-instance\" not found"})
			}),
			expectedError: "could not delete the autoscale on RPaaS API: 404 Not Found",
		},

		"when autoscale is removed": {
			args: []string{"autoscale", "remove", "-s", "my-service", "-i", "my-instance"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}),
			expected: "Autoscale of my-service/my-instance successfully removed\n",
		},
	}

	for _, serverGen := range AllRpaasAPIServerGenerators {
		t.Run("", func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					require.NotNil(t, tt.handler, "you must provide an HTTP handler")
					server, args := serverGen(t, tt.handler)
					defer server.Close()

					args = append(args, tt.args...)

					var stdout bytes.Buffer
					err := NewApp(&stdout, io.Discard, nil).Run(args)
					if tt.expectedError != "" {
						assert.EqualError(t, err, tt.expectedError)
						return
					}
					require.NoError(t, err)
					assert.Equal(t, tt.expected, stdout.String())
				})
			}
		})
	}
}

func TestUpdateAutoscale(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args          []string
		expected      string
		expectedError string
		handler       http.HandlerFunc
	}{
		"when instance doesn't exist": {
			args: []string{"autoscale", "update", "-s", "my-service", "-i", "my-instance", "--min", "0", "--max", "10", "--cpu", "75"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(autogenerated.Error{Msg: "instance \"my-instance\" not found"})
			}),
			expectedError: "could not update the autoscale on RPaaS API: 404 Not Found",
		},

		"when autoscale is successufully updated": {
			args: []string{"autoscale", "update", "-s", "my-service", "-i", "my-instance", "--min", "0", "--max", "10", "--cpu", "75"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var data map[string]any
				err := json.NewDecoder(r.Body).Decode(&data)
				require.NoError(t, err)
				assert.Equal(t, map[string]any{"minReplicas": float64(0), "maxReplicas": float64(10), "cpu": float64(75)}, data)

				w.WriteHeader(http.StatusNoContent)
			}),
			expected: "Autoscale of my-service/my-instance successfully updated!\n",
		},

		"with CPU + RPS scalers": {
			args: []string{"autoscale", "update", "-s", "my-service", "-i", "my-instance", "--min", "0", "--max", "10", "--cpu", "80", "--rps", "100"},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var data map[string]any
				err := json.NewDecoder(r.Body).Decode(&data)
				require.NoError(t, err)

				expected := map[string]any{
					"minReplicas": float64(0),
					"maxReplicas": float64(10),
					"cpu":         float64(80),
					"rps":         float64(100),
				}
				assert.Equal(t, expected, data)

				w.WriteHeader(http.StatusNoContent)
			}),
			expected: "Autoscale of my-service/my-instance successfully updated!\n",
		},

		"with schedules": {
			args: []string{"autoscale", "update", "-s", "my-service", "-i", "my-instance", "--min", "0", "--max", "10", "--schedule", `{"minReplicas": 1, "start": "00 08 * * 1-5", "end": "00 20 * * 1-5"}`, "--schedule", `{"minReplicas": 3, "start": "00 12 * * 1-5", "end": "00 13 * * 1-5"}`},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var data map[string]any
				err := json.NewDecoder(r.Body).Decode(&data)
				require.NoError(t, err)

				expected := map[string]any{
					"minReplicas": float64(0),
					"maxReplicas": float64(10),
					"schedules": []any{
						map[string]any{"minReplicas": float64(1), "start": "00 08 * * 1-5", "end": "00 20 * * 1-5"},
						map[string]any{"minReplicas": float64(3), "start": "00 12 * * 1-5", "end": "00 13 * * 1-5"},
					},
				}
				assert.Equal(t, expected, data)

				w.WriteHeader(http.StatusNoContent)
			}),
			expected: "Autoscale of my-service/my-instance successfully updated!\n",
		},
	}

	for _, serverGen := range AllRpaasAPIServerGenerators {
		t.Run("", func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					require.NotNil(t, tt.handler, "you must provide an HTTP handler")
					server, args := serverGen(t, tt.handler)
					defer server.Close()

					args = append(args, tt.args...)

					var stdout bytes.Buffer
					err := NewApp(&stdout, io.Discard, nil).Run(args)
					if tt.expectedError != "" {
						assert.EqualError(t, err, tt.expectedError)
						return
					}
					require.NoError(t, err)
					assert.Equal(t, tt.expected, stdout.String())
				})
			}
		})
	}
}
