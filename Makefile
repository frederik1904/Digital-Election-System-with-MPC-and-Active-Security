run: libs
	go run .

libs:
	go get "github.com/go-resty/resty"
	go get "github.com/google/uuid"
	go get "github.com/go-sql-driver/mysql"
	go get "github.com/pkg/errors"
	go get "golang.org/x/crypto/sha3"
	go get "github.com/stretchr/testify/assert"