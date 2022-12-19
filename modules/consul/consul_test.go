// SPDX-License-Identifier: GPL-3.0-or-later

package consul

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netdata/go.d.plugin/pkg/web"
)

var (
	datav1132Checks, _                        = os.ReadFile("testdata/v1.13.2/v1-agent-checks.json")
	dataV1132ClientSelf, _                    = os.ReadFile("testdata/v1.13.2/client_v1-agent-self.json")
	dataV1132ClientPromMetrics, _             = os.ReadFile("testdata/v1.13.2/client_v1-agent-metrics.txt")
	dataV1132ServerSelf, _                    = os.ReadFile("testdata/v1.13.2/server_v1-agent-self.json")
	dataV1132ServerSelfDisabledPrometheus, _  = os.ReadFile("testdata/v1.13.2/server_v1-agent-self_disabled_prom.json")
	dataV1132ServerSelfWithHostname, _        = os.ReadFile("testdata/v1.13.2/server_v1-agent-self_with_hostname.json")
	dataV1132ServerPromMetrics, _             = os.ReadFile("testdata/v1.13.2/server_v1-agent-metrics.txt")
	dataV1132ServerPromMetricsWithHostname, _ = os.ReadFile("testdata/v1.13.2/server_v1-agent-metrics_with_hostname.txt")
)

func Test_testDataIsValid(t *testing.T) {
	for name, data := range map[string][]byte{
		"datav1132Checks":                        datav1132Checks,
		"dataV1132ClientSelf":                    dataV1132ClientSelf,
		"dataV1132ClientPromMetrics":             dataV1132ClientPromMetrics,
		"dataV1132ServerSelf":                    dataV1132ServerSelf,
		"dataV1132ServerSelfWithHostname":        dataV1132ServerSelfWithHostname,
		"dataV1132ServerSelfDisabledPrometheus":  dataV1132ServerSelfDisabledPrometheus,
		"dataV1132ServerPromMetrics":             dataV1132ServerPromMetrics,
		"dataV1132ServerPromMetricsWithHostname": dataV1132ServerPromMetricsWithHostname,
	} {
		require.NotNilf(t, data, name)
	}
}

func TestConsul_Init(t *testing.T) {
	tests := map[string]struct {
		wantFail bool
		config   Config
	}{
		"success with default": {
			wantFail: false,
			config:   New().Config,
		},
		"fail when URL not set": {
			wantFail: true,
			config: Config{
				HTTP: web.HTTP{
					Request: web.Request{URL: ""},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			consul := New()
			consul.Config = test.config

			if test.wantFail {
				assert.False(t, consul.Init())
			} else {
				assert.True(t, consul.Init())
			}
		})
	}
}

func TestConsul_Check(t *testing.T) {
	tests := map[string]struct {
		wantFail bool
		prepare  func(t *testing.T) (consul *Consul, cleanup func())
	}{
		"success on response from Consul v1.13.2 server": {
			wantFail: false,
			prepare:  caseConsulV1132ServerResponse,
		},
		"success on response from Consul v1.13.2 server with enabled hostname": {
			wantFail: false,
			prepare:  caseConsulV1132ServerWithHostnameResponse,
		},
		"success on response from Consul v1.13.2 server with disabled prometheus": {
			wantFail: false,
			prepare:  caseConsulV1132ServerWithDisabledPrometheus,
		},
		"success on response from Consul v1.13.2 client": {
			wantFail: false,
			prepare:  caseConsulV1132ClientResponse,
		},
		"fail on invalid data response": {
			wantFail: true,
			prepare:  caseInvalidDataResponse,
		},
		"fail on connection refused": {
			wantFail: true,
			prepare:  caseConnectionRefused,
		},
		"fail on 404 response": {
			wantFail: true,
			prepare:  case404,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			consul, cleanup := test.prepare(t)
			defer cleanup()

			if test.wantFail {
				assert.False(t, consul.Check())
			} else {
				assert.True(t, consul.Check())
			}
		})
	}
}

func TestConsul_Collect(t *testing.T) {
	tests := map[string]struct {
		prepare         func(t *testing.T) (consul *Consul, cleanup func())
		wantNumOfCharts int
		wantMetrics     map[string]int64
	}{
		"success on response from Consul v1.13.2 server": {
			prepare:         caseConsulV1132ServerResponse,
			wantNumOfCharts: len(serverCharts) + 3 + 1, // 3 node, 1 service check
			wantMetrics: map[string]int64{
				"autopilot_failure_tolerance":               1,
				"autopilot_healthy_no":                      0,
				"autopilot_healthy_yes":                     1,
				"client_rpc":                                6838,
				"client_rpc_exceeded":                       0,
				"client_rpc_failed":                         0,
				"health_check_chk1_critical_status":         0,
				"health_check_chk1_maintenance_status":      0,
				"health_check_chk1_passing_status":          1,
				"health_check_chk1_warning_status":          0,
				"health_check_chk2_critical_status":         1,
				"health_check_chk2_maintenance_status":      0,
				"health_check_chk2_passing_status":          0,
				"health_check_chk2_warning_status":          0,
				"health_check_chk3_critical_status":         1,
				"health_check_chk3_maintenance_status":      0,
				"health_check_chk3_passing_status":          0,
				"health_check_chk3_warning_status":          0,
				"health_check_mysql_critical_status":        1,
				"health_check_mysql_maintenance_status":     0,
				"health_check_mysql_passing_status":         0,
				"health_check_mysql_warning_status":         0,
				"kvs_apply_count":                           0,
				"kvs_apply_quantile=0.5":                    0,
				"kvs_apply_quantile=0.9":                    0,
				"kvs_apply_quantile=0.99":                   0,
				"kvs_apply_sum":                             0,
				"raft_apply":                                10681000,
				"raft_boltdb_freelistBytes":                 11264,
				"raft_boltdb_logsPerBatch_count":            12360,
				"raft_boltdb_logsPerBatch_quantile=0.5":     1000000,
				"raft_boltdb_logsPerBatch_quantile=0.9":     1000000,
				"raft_boltdb_logsPerBatch_quantile=0.99":    1000000,
				"raft_boltdb_logsPerBatch_sum":              12362000,
				"raft_boltdb_storeLogs_count":               12360,
				"raft_boltdb_storeLogs_quantile=0.5":        13176624,
				"raft_boltdb_storeLogs_quantile=0.9":        13176624,
				"raft_boltdb_storeLogs_quantile=0.99":       13176624,
				"raft_boltdb_storeLogs_sum":                 651888027,
				"raft_commitTime_count":                     12345,
				"raft_commitTime_quantile=0.5":              41146488,
				"raft_commitTime_quantile=0.9":              41146488,
				"raft_commitTime_quantile=0.99":             41146488,
				"raft_commitTime_sum":                       955781149,
				"raft_fsm_lastRestoreDuration":              2,
				"raft_leader_lastContact_count":             80917,
				"raft_leader_lastContact_quantile=0.5":      33000000,
				"raft_leader_lastContact_quantile=0.9":      68000000,
				"raft_leader_lastContact_quantile=0.99":     68000000,
				"raft_leader_lastContact_sum":               3066900000,
				"raft_leader_oldestLogAge":                  166046464,
				"raft_rpc_installSnapshot_count":            0,
				"raft_rpc_installSnapshot_quantile=0.5":     0,
				"raft_rpc_installSnapshot_quantile=0.9":     0,
				"raft_rpc_installSnapshot_quantile=0.99":    0,
				"raft_rpc_installSnapshot_sum":              0,
				"raft_state_candidate":                      1,
				"raft_state_leader":                         1,
				"raft_thread_fsm_saturation_count":          11923,
				"raft_thread_fsm_saturation_quantile=0.5":   0,
				"raft_thread_fsm_saturation_quantile=0.9":   0,
				"raft_thread_fsm_saturation_quantile=0.99":  0,
				"raft_thread_fsm_saturation_sum":            90,
				"raft_thread_main_saturation_count":         43067,
				"raft_thread_main_saturation_quantile=0.5":  0,
				"raft_thread_main_saturation_quantile=0.9":  0,
				"raft_thread_main_saturation_quantile=0.99": 0,
				"raft_thread_main_saturation_sum":           205409,
				"runtime_alloc_bytes":                       53065368,
				"runtime_sys_bytes":                         84955160,
				"runtime_total_gc_pause_ns":                 1372001280,
				"server_isLeader_no":                        0,
				"server_isLeader_yes":                       1,
				"txn_apply_count":                           0,
				"txn_apply_quantile=0.5":                    0,
				"txn_apply_quantile=0.9":                    0,
				"txn_apply_quantile=0.99":                   0,
				"txn_apply_sum":                             0,
			},
		},
		"success on response from Consul v1.13.2 server with enabled hostname": {
			prepare:         caseConsulV1132ServerResponse,
			wantNumOfCharts: len(serverCharts) + 3 + 1, // 3 node, 1 service check
			wantMetrics: map[string]int64{
				"autopilot_failure_tolerance":               1,
				"autopilot_healthy_no":                      0,
				"autopilot_healthy_yes":                     1,
				"client_rpc":                                6838,
				"client_rpc_exceeded":                       0,
				"client_rpc_failed":                         0,
				"health_check_chk1_critical_status":         0,
				"health_check_chk1_maintenance_status":      0,
				"health_check_chk1_passing_status":          1,
				"health_check_chk1_warning_status":          0,
				"health_check_chk2_critical_status":         1,
				"health_check_chk2_maintenance_status":      0,
				"health_check_chk2_passing_status":          0,
				"health_check_chk2_warning_status":          0,
				"health_check_chk3_critical_status":         1,
				"health_check_chk3_maintenance_status":      0,
				"health_check_chk3_passing_status":          0,
				"health_check_chk3_warning_status":          0,
				"health_check_mysql_critical_status":        1,
				"health_check_mysql_maintenance_status":     0,
				"health_check_mysql_passing_status":         0,
				"health_check_mysql_warning_status":         0,
				"kvs_apply_count":                           0,
				"kvs_apply_quantile=0.5":                    0,
				"kvs_apply_quantile=0.9":                    0,
				"kvs_apply_quantile=0.99":                   0,
				"kvs_apply_sum":                             0,
				"raft_apply":                                10681000,
				"raft_boltdb_freelistBytes":                 11264,
				"raft_boltdb_logsPerBatch_count":            12360,
				"raft_boltdb_logsPerBatch_quantile=0.5":     1000000,
				"raft_boltdb_logsPerBatch_quantile=0.9":     1000000,
				"raft_boltdb_logsPerBatch_quantile=0.99":    1000000,
				"raft_boltdb_logsPerBatch_sum":              12362000,
				"raft_boltdb_storeLogs_count":               12360,
				"raft_boltdb_storeLogs_quantile=0.5":        13176624,
				"raft_boltdb_storeLogs_quantile=0.9":        13176624,
				"raft_boltdb_storeLogs_quantile=0.99":       13176624,
				"raft_boltdb_storeLogs_sum":                 651888027,
				"raft_commitTime_count":                     12345,
				"raft_commitTime_quantile=0.5":              41146488,
				"raft_commitTime_quantile=0.9":              41146488,
				"raft_commitTime_quantile=0.99":             41146488,
				"raft_commitTime_sum":                       955781149,
				"raft_fsm_lastRestoreDuration":              2,
				"raft_leader_lastContact_count":             80917,
				"raft_leader_lastContact_quantile=0.5":      33000000,
				"raft_leader_lastContact_quantile=0.9":      68000000,
				"raft_leader_lastContact_quantile=0.99":     68000000,
				"raft_leader_lastContact_sum":               3066900000,
				"raft_leader_oldestLogAge":                  166046464,
				"raft_rpc_installSnapshot_count":            0,
				"raft_rpc_installSnapshot_quantile=0.5":     0,
				"raft_rpc_installSnapshot_quantile=0.9":     0,
				"raft_rpc_installSnapshot_quantile=0.99":    0,
				"raft_rpc_installSnapshot_sum":              0,
				"raft_state_candidate":                      1,
				"raft_state_leader":                         1,
				"raft_thread_fsm_saturation_count":          11923,
				"raft_thread_fsm_saturation_quantile=0.5":   0,
				"raft_thread_fsm_saturation_quantile=0.9":   0,
				"raft_thread_fsm_saturation_quantile=0.99":  0,
				"raft_thread_fsm_saturation_sum":            90,
				"raft_thread_main_saturation_count":         43067,
				"raft_thread_main_saturation_quantile=0.5":  0,
				"raft_thread_main_saturation_quantile=0.9":  0,
				"raft_thread_main_saturation_quantile=0.99": 0,
				"raft_thread_main_saturation_sum":           205409,
				"runtime_alloc_bytes":                       53065368,
				"runtime_sys_bytes":                         84955160,
				"runtime_total_gc_pause_ns":                 1372001280,
				"server_isLeader_no":                        0,
				"server_isLeader_yes":                       1,
				"txn_apply_count":                           0,
				"txn_apply_quantile=0.5":                    0,
				"txn_apply_quantile=0.9":                    0,
				"txn_apply_quantile=0.99":                   0,
				"txn_apply_sum":                             0,
			},
		},
		"success on response from Consul v1.13.2 server with disabled prometheus": {
			prepare:         caseConsulV1132ServerWithDisabledPrometheus,
			wantNumOfCharts: 3 + 1, // 3 node, 1 service check
			wantMetrics: map[string]int64{
				"health_check_chk1_critical_status":     0,
				"health_check_chk1_maintenance_status":  0,
				"health_check_chk1_passing_status":      1,
				"health_check_chk1_warning_status":      0,
				"health_check_chk2_critical_status":     1,
				"health_check_chk2_maintenance_status":  0,
				"health_check_chk2_passing_status":      0,
				"health_check_chk2_warning_status":      0,
				"health_check_chk3_critical_status":     1,
				"health_check_chk3_maintenance_status":  0,
				"health_check_chk3_passing_status":      0,
				"health_check_chk3_warning_status":      0,
				"health_check_mysql_critical_status":    1,
				"health_check_mysql_maintenance_status": 0,
				"health_check_mysql_passing_status":     0,
				"health_check_mysql_warning_status":     0,
			},
		},
		"success on response from Consul v1.13.2 client": {
			prepare:         caseConsulV1132ClientResponse,
			wantNumOfCharts: len(clientCharts) + 3 + 1, // 3 node, 1 service check
			wantMetrics: map[string]int64{
				"client_rpc":                            34,
				"client_rpc_exceeded":                   0,
				"client_rpc_failed":                     0,
				"health_check_chk1_critical_status":     0,
				"health_check_chk1_maintenance_status":  0,
				"health_check_chk1_passing_status":      1,
				"health_check_chk1_warning_status":      0,
				"health_check_chk2_critical_status":     1,
				"health_check_chk2_maintenance_status":  0,
				"health_check_chk2_passing_status":      0,
				"health_check_chk2_warning_status":      0,
				"health_check_chk3_critical_status":     1,
				"health_check_chk3_maintenance_status":  0,
				"health_check_chk3_passing_status":      0,
				"health_check_chk3_warning_status":      0,
				"health_check_mysql_critical_status":    1,
				"health_check_mysql_maintenance_status": 0,
				"health_check_mysql_passing_status":     0,
				"health_check_mysql_warning_status":     0,
				"runtime_alloc_bytes":                   26333408,
				"runtime_sys_bytes":                     51201032,
				"runtime_total_gc_pause_ns":             4182423,
			},
		},
		"fail on invalid data response": {
			prepare:         caseInvalidDataResponse,
			wantNumOfCharts: 0,
			wantMetrics:     nil,
		},
		"fail on connection refused": {
			prepare:         caseConnectionRefused,
			wantNumOfCharts: 0,
			wantMetrics:     nil,
		},
		"fail on 404 response": {
			prepare:         case404,
			wantNumOfCharts: 0,
			wantMetrics:     nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			consul, cleanup := test.prepare(t)
			defer cleanup()

			mx := consul.Collect()

			require.Equal(t, test.wantMetrics, mx)
			if len(test.wantMetrics) > 0 {
				assert.Equal(t, test.wantNumOfCharts, len(*consul.Charts()))
			}
		})
	}
}

func caseConsulV1132ServerResponse(t *testing.T) (*Consul, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == urlPathAgentSelf:
				_, _ = w.Write(dataV1132ServerSelf)
			case r.URL.Path == urlPathAgentChecks:
				_, _ = w.Write(datav1132Checks)
			case r.URL.Path == urlPathAgentMetrics && r.URL.RawQuery == "format=prometheus":
				_, _ = w.Write(dataV1132ServerPromMetrics)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

	consul := New()
	consul.URL = srv.URL

	require.True(t, consul.Init())

	return consul, srv.Close
}

func caseConsulV1132ServerWithHostnameResponse(t *testing.T) (*Consul, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == urlPathAgentSelf:
				_, _ = w.Write(dataV1132ServerSelfWithHostname)
			case r.URL.Path == urlPathAgentChecks:
				_, _ = w.Write(datav1132Checks)
			case r.URL.Path == urlPathAgentMetrics && r.URL.RawQuery == "format=prometheus":
				_, _ = w.Write(dataV1132ServerPromMetricsWithHostname)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

	consul := New()
	consul.URL = srv.URL

	require.True(t, consul.Init())

	return consul, srv.Close
}

func caseConsulV1132ServerWithDisabledPrometheus(t *testing.T) (*Consul, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == urlPathAgentSelf:
				_, _ = w.Write(dataV1132ServerSelfDisabledPrometheus)
			case r.URL.Path == urlPathAgentChecks:
				_, _ = w.Write(datav1132Checks)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

	consul := New()
	consul.URL = srv.URL

	require.True(t, consul.Init())

	return consul, srv.Close
}

func caseConsulV1132ClientResponse(t *testing.T) (*Consul, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == urlPathAgentSelf:
				_, _ = w.Write(dataV1132ClientSelf)
			case r.URL.Path == urlPathAgentChecks:
				_, _ = w.Write(datav1132Checks)
			case r.URL.Path == urlPathAgentMetrics && r.URL.RawQuery == "format=prometheus":
				_, _ = w.Write(dataV1132ClientPromMetrics)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

	consul := New()
	consul.URL = srv.URL

	require.True(t, consul.Init())

	return consul, srv.Close
}

func caseInvalidDataResponse(t *testing.T) (*Consul, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("hello and\n goodbye"))
		}))

	consul := New()
	consul.URL = srv.URL

	require.True(t, consul.Init())

	return consul, srv.Close
}

func caseConnectionRefused(t *testing.T) (*Consul, func()) {
	t.Helper()
	consul := New()
	consul.URL = "http://127.0.0.1:65535/"
	require.True(t, consul.Init())

	return consul, func() {}
}

func case404(t *testing.T) (*Consul, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

	consul := New()
	consul.URL = srv.URL
	require.True(t, consul.Init())

	return consul, srv.Close
}
