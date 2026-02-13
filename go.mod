module github.com/rhedin/Abe_eliasdb

go 1.25

require (
	github.com/gorilla/websocket v1.4.1
	github.com/rhedin/Abe_common v1.5.2
	github.com/rhedin/Abe_ecal v1.6.3
	github.com/rhedin/Abe_editor v0.0.0-20260212225356-56e22903e533
)

replace (
	github.com/rhedin/Abe_common => ../Abe_common
	github.com/rhedin/Abe_ecal => ../Abe_ecal
	github.com/rhedin/Abe_editor => ../Abe_editor
)
