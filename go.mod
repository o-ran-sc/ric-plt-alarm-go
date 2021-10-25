module gerrit.o-ran-sc.org/r/ric-plt/alarm-go

go 1.13

replace gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm => ./alarm/

replace gerrit.o-ran-sc.org/r/ric-plt/xapp-frame => gerrit.o-ran-sc.org/r/ric-plt/xapp-frame.git v0.9.2-1

replace gerrit.o-ran-sc.org/r/ric-plt/sdlgo => gerrit.o-ran-sc.org/r/ric-plt/sdlgo.git v0.7.0

replace gerrit.o-ran-sc.org/r/com/golog => gerrit.o-ran-sc.org/r/com/golog.git v0.0.2

require (
	gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm v0.0.0-00010101000000-000000000000
	gerrit.o-ran-sc.org/r/ric-plt/xapp-frame v0.0.0-00010101000000-000000000000
	github.com/go-openapi/runtime v0.19.11
	github.com/go-openapi/strfmt v0.19.4
	github.com/gorilla/mux v1.7.1
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/prometheus/alertmanager v0.20.0
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	github.com/thatisuday/commando v1.0.4
)
