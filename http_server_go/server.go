package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		log.Println("Connected to port 4221")
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go processConnection(conn)
	}
}

type Request struct {
	method      string
	scheme      string
	server      string
	path        string
	queryString string
	headers     map[string]string
	body        string
}

func NewRequest(rawRequest []byte) *Request {
	r := Request{}
	rawString := string(rawRequest)
	lines := strings.Split(rawString, "\r\n")
	fields := strings.Fields(lines[0])
	r.method = fields[0]
	r.path = fields[1]

	//headers
	headers := make(map[string]string)
	headerStrings := lines[1:]
	for _, line := range headerStrings {
		splitLine := strings.Split(line, ": ")
		if len(splitLine) == 2 {
			headers[splitLine[0]] = splitLine[1]
		} else {
			if len(line) > 0 {
				r.body = line
			}
		}
	}
	r.headers = headers

	return &r
}

func (r *Request) GetHeader(header string) (val string, ok bool) {
	val, ok = r.headers[header]
	return
}

func (r *Request) GetLastPartOfPath() string {
	result := strings.Split(r.path, "/")
	if len(result) > 0 {
		return result[len(result)-1]
	}
	return ""
}

func NewResponseString(statusCode int, headers map[string]string, body string) []byte {
	if headers == nil {
		headers = make(map[string]string)
	}
	protocol := "HTTP/1.1"

	var statusLine string
	switch statusCode {
	case 200:
		statusLine = "200 OK"
	case 404:
		statusLine = "404 Not Found"
	case 405:
		statusLine = "405 Method Not Allowed"
	case 201:
		statusLine = "201 Created"
	default:
		statusLine = "500 Internal Server Error"
	}
	protocolLine := fmt.Sprintf("%s %s\r\n", protocol, statusLine)

	var headerString string
	if len(headers) > 0 {
		for k, v := range headers {
			headerString += fmt.Sprintf("%s: %s\r\n", k, v)
		}
	}
	if len(body) > 0 {
		headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	}
	var headersLine string
	if len(headers) > 0 {
		for k, v := range headers {
			headersLine += fmt.Sprintf("%s: %s\r\n", k, v)
		}
		headersLine += "\r\n"
	}
	return []byte(fmt.Sprintf("%s%s%s", protocolLine, headersLine, body) + "\r\n\r\n")
}

func extractFileDirInfo(req *Request) (string, error) {
	filename := req.GetLastPartOfPath()
	if len(os.Args) == 3 {
		dir := os.Args[2]
		return dir + filename, nil
	}
	return "", fmt.Errorf("bad diretory information")
}

func handleRoot() []byte {
	return NewResponseString(200, nil, "")
}

func handleFiles(req *Request) []byte {
	if req.method == "GET" {
		responseHeaders := make(map[string]string)

		fullPath, err := extractFileDirInfo(req)
		if err != nil {
			return NewResponseString(404, nil, "")
		} else {
			if _, err := os.Stat(fullPath); err == nil {
				file, _ := os.Open(fullPath)
				contents, _ := io.ReadAll(file)
				responseHeaders["Content-Type"] = "application/octet-stream"
				return NewResponseString(200, responseHeaders, string(contents))
			}
		}
		return NewResponseString(404, nil, "")
	} else if req.method == "POST" {
		filename := req.GetLastPartOfPath()
		if len(os.Args) == 3 {
			dir := os.Args[2]
			fullPath := dir + filename
			err := os.WriteFile(fullPath, []byte(req.body), 0644)
			if err != nil {
				return NewResponseString(500, nil, "")
			} else {
				return NewResponseString(201, nil, "")
			}
		} else {
			return NewResponseString(404, nil, "")
		}
	}
	return NewResponseString(405, nil, "")
}

func processConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalln("Error reading: ", err.Error())
	}

	responseHeaders := make(map[string]string)

	req := NewRequest(buf[:n])

	var response []byte

	if req.path == "/" {
		response = handleRoot()
	} else if strings.Contains(req.path, "/files/") {
		response = handleFiles(req)
	} else if strings.Contains(req.path, "/user-agent") {
		val, _ := req.GetHeader("User-Agent")
		responseHeaders["Content-Type"] = "text/plain"
		response = NewResponseString(200, responseHeaders, val)

	} else if !strings.Contains(req.path, "/echo/") {
		response = NewResponseString(404, nil, "")
	} else {
		randomString := req.GetLastPartOfPath()
		responseHeaders["Content-Type"] = "text/plain"
		response = NewResponseString(200, responseHeaders, randomString)
	}
	if _, err := conn.Write(response); err != nil {
		log.Fatalln("Error writing to connection: ", err.Error())
	}
}
