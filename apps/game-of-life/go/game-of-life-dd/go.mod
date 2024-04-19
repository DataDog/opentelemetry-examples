module github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/game-of-life-dd

go 1.19

require (
	github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb v0.0.0-20230327194142-fd42f2429870
	github.com/golang/mock v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/pkg/errors v0.9.1
	go.uber.org/zap v1.24.0
	google.golang.org/grpc v1.56.3
	gopkg.in/DataDog/dd-trace-go.v1 v1.39.0-alpha.1.0.20221115095721-0f06e4e80273
)

require (
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.0.0-20211129110424-6491aa3bf583 // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.40.0-rc.2.0.20221012133526-043d202b24fc // indirect
	github.com/DataDog/datadog-go v4.8.3+incompatible // indirect
	github.com/DataDog/datadog-go/v5 v5.1.1 // indirect
	github.com/DataDog/sketches-go v1.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/golang/glog v1.1.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.4.0 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	github.com/theupdateframework/go-tuf v0.3.2 // indirect
	github.com/tinylib/msgp v1.1.6 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/goleak v1.2.1 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go4.org/intern v0.0.0-20220617035311-6925f38cc365 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20220617031537-928513b29760 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.2.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	inet.af/netaddr v0.0.0-20220617031823-097006376321 // indirect
)

replace github.com/DataDog/opentelemetry-examples/apps/game-of-life/go/pb => ../pb/
