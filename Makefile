run: libs
	go run .

libs:
	go get "github.com/go-resty/resty"
	go get "github.com/google/uuid"
	go get "github.com/go-sql-driver/mysql"
	go get "github.com/glendc/go-external-ip"