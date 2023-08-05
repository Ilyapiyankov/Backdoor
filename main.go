package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var clients = make(map[string]net.Conn)
var cwd = make(map[string]string)
var mutex = &sync.Mutex{}

func handleConnection(conn net.Conn) {
	username := conn.RemoteAddr().String()
	clients[username] = conn

	defer func() {
		conn.Close()
		mutex.Lock()
		delete(clients, username)
		delete(cwd, username)
		mutex.Unlock()
	}()

	log.Printf("New connection from: %s", username)

	scanner := bufio.NewScanner(conn)

	io.WriteString(conn, fmt.Sprintf("Welcome to the server, %s!\n", username))

	for {
		io.WriteString(conn, fmt.Sprintf("%s> ", cwd[username]))

		if !scanner.Scan() {
			break
		}

		cmd := scanner.Text()

		cmdParts := strings.Fields(cmd)

		switch cmdParts[0] {
		case "exit":
			return
		case "cd":
			if len(cmdParts) < 2 {
				io.WriteString(conn, "Usage: cd [directory]\n"+cmdParts[0]+" "+cmdParts[1])
				continue
			}

			dir := cmdParts[1][:len(cmdParts[1])]
			err := os.Chdir(dir)
			if err != nil {
				io.WriteString(conn, fmt.Sprintf("Failed to change directory: %s. dir:%s is incorrect\n", err.Error(), dir))
				continue
			}

			dir, err = os.Getwd()
			if err != nil {
				io.WriteString(conn, fmt.Sprintf("Failed to get current directory: %s\n", err.Error()))
				continue
			}

			cwd[username] = dir
		default:
			command := exec.Command(cmdParts[0], cmdParts[1:]...)
			command.Dir = cwd[username]

			output, err := command.CombinedOutput()
			if err != nil {
				io.WriteString(conn, fmt.Sprintf("Command execution failed: %s\n", err.Error()))
				continue
			}

			io.WriteString(conn, string(output))
		}
	}

	log.Printf("Connection closed for: %s", username)
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
	defer listener.Close()

	log.Println("Server started on localhost:8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %s", err)
			continue
		}

		mutex.Lock()
		cwd[conn.RemoteAddr().String()], _ = os.Getwd()
		mutex.Unlock()

		go handleConnection(conn)
	}
}
