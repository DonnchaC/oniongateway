package main

import (
    "net"
    "testing"
)

func TestNetCopy(t *testing.T) {
    listener, err := net.Listen("tcp", "127.0.0.1:0") // pick free port
    if err != nil {
        t.Fatal(err)
    }
    defer listener.Close()
    go func() {
        conn, err := listener.Accept()
        if err != nil {
            t.Fatal(err)
        }
        defer conn.Close()
        // write
        for i := 0; i < 10000; i++ {
            conn.Write([]byte("AAA"))
        }
        // read and check
        buffer := make([]byte, 3)
        for i := 0; i < 10000; i++ {
            bytesRead, err := conn.Read(buffer)
            if err != nil {
                t.Fatal(err)
            }
            if bytesRead != 3 {
                t.Fatalf("Mismatch in bytesRead. Expected %d, got %d", 3, bytesRead)
            }
            if string(buffer) != "AAA" {
                t.Fatalf("Mismatch. Expected %q, got %q", "AAA", buffer)
            }
        }
    }()
    // echo client
    client, err := net.Dial("tcp", listener.Addr().String())
    if err != nil {
        t.Fatal(err)
    }
    finished := make(chan struct{})
    go netCopy(client, client, finished)
    <-finished
}
