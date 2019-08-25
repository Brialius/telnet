package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"sync"
	"time"
)

const connectTimeout = 5 * time.Second

var (
	timeoutInt int
	timeout    time.Duration
	timeStart  = time.Now()
)

func init() {
	flag.IntVar(&timeoutInt, "timeout", 60, "Connection timeout in seconds")
	flag.Parse()
	timeout = time.Duration(timeoutInt) * time.Second

	//Handle SIGINT (Ctrl-C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		printAndExit("\nGot SIGINT\n", 1)
	}()
}

func main() {

	args := flag.Args()

	if len(args) < 2 {
		printAndExit(fmt.Sprintf("Usage of %s:\nADDRESS PORT [-timeout (default 60)]\n", path.Base(os.Args[0])), 1)
	}

	address := args[0]
	port := args[1]

	fmt.Printf("Trying %s:%s with timeout %v...\n", address, port, timeout)
	dialer := net.Dialer{
		Timeout: connectTimeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", address, port))

	if err != nil {
		printAndExit(fmt.Sprintf("Can't connect: %v\n", err), 1)
	}

	fmt.Printf("Connected to %s:%s.\n", address, port)

	defer func() {
		if err := conn.Close(); err != nil {
			printAndExit(fmt.Sprintf("Can't close connection: %v", err), 1)
		}
	}()

	input := scanToChan(os.Stdin, cancel)
	output := scanToChan(conn, cancel)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		readRoutine(ctx, output)
		cancel()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		writeRoutine(ctx, conn, input)
		cancel()
		wg.Done()
	}()

	wg.Wait()
	printAndExit("Connection closed.", 0)
}

func scanToChan(in io.Reader, cancel context.CancelFunc) chan string {
	c := make(chan string, 1)
	go func() {
		readScan(in, c)
		cancel()
	}()
	return c
}

func readScan(in io.Reader, c chan<- string) {
	input := bufio.NewScanner(in)
	for {
		if !input.Scan() {
			break
		}
		c <- input.Text()
	}
}

func writeRoutine(ctx context.Context, out io.Writer, c <-chan string) {
OUTER:
	for {
		select {
		case <-ctx.Done():
			break OUTER
		case line, ok := <-c:
			if !ok {
				fmt.Println("Write error: channel is closed")
				break OUTER
			}
			_, err := out.Write([]byte(line + "\n"))
			if err != nil {
				fmt.Printf("Wrire error: %s\n", err)
				break OUTER
			}
		}
	}
}

func readRoutine(ctx context.Context, c <-chan string) {
OUTER:
	for {
		select {
		case <-ctx.Done():
			break OUTER
		case line, ok := <-c:
			if !ok {
				fmt.Println("Read error: channel is closed")
				break OUTER
			}
			fmt.Printf("%s\n", line)
		}
	}
}

func printAndExit(s string, e int) {
	fmt.Println(s)
	fmt.Printf("Stopped after %v\n", time.Since(timeStart))
	os.Exit(e)
}
