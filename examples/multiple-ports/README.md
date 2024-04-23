# multiple-ports

Quick & dirty project to test the **seelf** ability to handle multiple ports of specific types and check how they are managed by traefik.

## Usage

```sh
go run main.go -web 8080,8081 -tcp 9854,9855 -udp 9856,9857
```

## How to test

Quick & dirty ways to test if the server is actually listening.

```sh
curl http://localhost:8080 # HTTP
curl telnet://localhost:9855 <<< text # TCP
nc -v localhost 9855 # Other way for TCP
nc -v -u localhost 9856 # UDP
```
