package scripts

const (
  VTGateStart = `eval exec /vt/bin/vtgate $(cat <<END_OF_COMMAND
  -cell={{ .Cell.Name }}
  -logtostderr=true
  -stderrthreshold=0
  -port=15001
  -grpc_port=15991
  -service_map="grpc-vtgateservice"
  -cells_to_watch="{{ .Cell.Name }}"
  -tablet_types_to_wait="MASTER,REPLICA"
  -gateway_implementation="discoverygateway"
  -mysql_server_version="5.5.10-Vitess"
  {{ if eq .GlobalLockserver.Spec.Type "etcd2" }}
  -topo_implementation="etcd2"
  -topo_global_server_address="{{ .GlobalLockserver.Spec.Etcd2.Address }}"
  -topo_global_root={{ .GlobalLockserver.Spec.Etcd2.Path }}
  {{ end }}
  {{ if .Cell.Spec.MySQLProtocol }}
  -mysql_server_port=3306
  {{ if .Cell.Spec.MySQLProtocol.PasswordSecretRef }}
  -mysql_auth_server_impl="static"
  -mysql_auth_server_static_file="/mysqlcreds/creds.json"
  {{ else if eq .Cell.Spec.MySQLProtocol.AuthType "none" }}
  -mysql_auth_server_impl="none"
  {{ end }}
  {{ end }}
END_OF_COMMAND
)
`
)
