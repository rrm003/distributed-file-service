# distributed-file-service

A robust distributed system that provides file management services (upload, download, delete, rename) and real-time client-server synchronization using gRPC and Go. Includes computation services for basic operations like addition and sorting. Implements both synchronous and asynchronous communication patterns, leveraging Go's concurrency features.

Here's a README file in Markdown format based on your project report. It's structured and formatted for clarity and ease of copying:

---

# Distributed System Application

## Introduction

The objective of this project is to develop a robust **file management service** and a **computation service** using **remote procedure call (RPC)** based communication. The project focuses on:

-   File management techniques
-   Client-server communication
-   Asynchronous communication between client and server by implementing simple addition and sorting compute services

---

## Tools

-   **Programming Language**: Go
-   **Libraries**:
    -   `grpc`
    -   `sync`
    -   `fsnotify`
    -   `cobra`

---

## Implementation

### Part 1: File Management

Supports the following operations:

### 01. Start Server

![Start Server](images/01-start-server.png)

### 02. Upload Operation

![Upload Operation](images/02-upload-operation.png)

### 03. Download Operation

![Download Operation](images/03-download-operation.png)

### 04. Rename Operation

![Rename Operation](images/04-rename-operation.png)

### 05. Delete Operation

![Delete Operation](images/05-delete-operation.png)

### Part 2: File Synchronization

Synchronizes client-side changes with the server.

---

### 01. Start both server and client.

![Start Server and Client](images/11-start-server-client.png)

### 02. Create a file at client's side, it gets created to server side and saved to downloads folder

![Create a file at client's side](images/12-create-file-at-client.png)

### 03. Any update made to the client's side gets synced to respective file on the servers end.

![Update the file at client's end](images/13-update-file-at-client-end.png)

### 04. File renamed on the client side is also renamed to server's end.

![Rename file Operation](images/14-rename-operation.png)

### 05. If file deteled on the clients side is also deleted from the server.

![Delete file Operation](images/15-delete-operation.png)

## Directory Structure

```
.
├── client
│   ├── main1.go
│   └── main.go
├── downloads
│   └── serverfile.txt
├── file_management
│   ├── file_management_grpc.pb.go
│   ├── file_management.pb.go
│   └── file_management.proto
├── go.mod
├── go.sum
├── server
│   └── main.go
├── sync.go
├── test-server-file.txt
├── uploads
│   └── serverfile.txt
```

---

## Learnings

### 1. gRPC

-   Implementing the project using **gRPC** provided valuable insights into modern RPC frameworks.
-   Key concepts learned:
    -   Service definition using **Protocol Buffers**.

### 2. Concurrency

-   Explored Go's concurrency model using **goroutines**.
-   Learned about lightweight thread management and idiomatic patterns for concurrent programming.

### 3. File Operations

-   Used the **fsnotify** package for tracking file operations.
-   Gained experience with event-driven programming paradigms.
-   Practiced efficient file handling using Go's standard library.

### 4. Synchronous and Asynchronous APIs

-   Implemented both **sync** and **async** APIs using Go's concurrency features.
-   Designed efficient and responsive APIs by leveraging:
    -   **Goroutines**
    -   **Channels**
