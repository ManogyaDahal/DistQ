module github.com/ManogyaDahal/DistQ/test_app

go 1.26.1

require (
	github.com/ManogyaDahal/DistQ v0.0.0
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/redis/go-redis/v9 v9.19.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

replace github.com/ManogyaDahal/DistQ => ../
