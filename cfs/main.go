package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/context"

	"github.com/codegangsta/cli"
	pb "github.com/creiht/formic/proto"
	mb "github.com/letterj/oohhc/proto/filesystem"

	"bazil.org/fuse"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type server struct {
	fs *fs
	wg sync.WaitGroup
}

func newserver(fs *fs) *server {
	s := &server{
		fs: fs,
	}
	return s
}

func (s *server) serve() error {
	defer s.wg.Wait()

	for {
		req, err := s.fs.conn.ReadRequest()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.fs.handle(req)
		}()
	}
	return nil
}

func debuglog(msg interface{}) {
	fmt.Fprintf(os.Stderr, "%v\n", msg)
}

type rpc struct {
	conn *grpc.ClientConn
	api  pb.ApiClient
}

func newrpc(conn *grpc.ClientConn) *rpc {
	r := &rpc{
		conn: conn,
		api:  pb.NewApiClient(conn),
	}

	return r
}

// NullWriter ...
type NullWriter int

func (NullWriter) Write([]byte) (int, error) { return 0, nil }

func main() {

	// Process command line arguments
	var token string
	var acctNum string
	var fsNum string
	var serverAddr string

	app := cli.NewApp()
	app.Name = "cfs"
	app.Usage = "Client used to test filesysd"
	app.Version = "0.5.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "token, T",
			Value:       "",
			Usage:       "Access token",
			EnvVar:      "OOHHC_TOKEN_KEY",
			Destination: &token,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "show",
			Usage: "Show a File Systems",
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for show.")
					os.Exit(1)
				}
				if token == "" {
					fmt.Println("Token is required")
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					panic(err)
				}
				fmt.Println(u.Scheme)
				acctNum = u.Host
				fsNum = u.Path[1:]
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.ShowFS(context.Background(), &mb.ShowFSRequest{Acctnum: acctNum, FSid: fsNum, Token: token})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
				log.Printf("SHOW Results: %s", result.Payload)
			},
		},
		{
			Name:  "create",
			Usage: "Create a File Systems",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, N",
					Value: "",
					Usage: "Name of the file system",
				},
			},
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for show.")
					os.Exit(1)
				}
				if token == "" {
					fmt.Println("Token is required")
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Printf("Url parse error: %v", err)
					os.Exit(1)
				}
				if u.Scheme == "aio" {
					serverAddr = "127.0.0.1:8448"
				} else if u.Scheme == "iad" {
					serverAddr = "api.ea.iad3.rackfs.com:8443"
				} else {
					fmt.Printf("Invalid region %s", u.Scheme)
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				acctNum = u.Host
				if u.Path != "" {
					fmt.Println("Invalid url scheme")
					os.Exit(1)
				}
				if c.String("name") == "" {
					fmt.Println("File system name is a required field.")
					os.Exit(1)
				}
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.CreateFS(context.Background(), &mb.CreateFSRequest{Acctnum: acctNum, FSName: c.String("name"), Token: token})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
				log.Printf("Create Results: %s", result.Payload)
			},
		},
		{
			Name:  "list",
			Usage: "List File Systems for an account",
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for list.")
					os.Exit(1)
				}
				if token == "" {
					fmt.Println("Token is required")
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Println("Invalid url scheme")
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				if u.Scheme == "aio" {
					serverAddr = "127.0.0.1:8448"
				} else if u.Scheme == "iad" {
					serverAddr = "api.ea.iad3.rackfs.com:8443"
				} else {
					fmt.Printf("Invalid region %s", u.Scheme)
					os.Exit(1)
				}
				acctNum = u.Host
				if u.Path != "" {
					fmt.Println("Invaid url")
					os.Exit(1)
				}
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.ListFS(context.Background(), &mb.ListFSRequest{Acctnum: acctNum, Token: token})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
				log.Printf("LIST Results: %s", result.Payload)
			},
		},
		{
			Name:  "delete",
			Usage: "Delete a File Systems",
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for delete.")
					os.Exit(1)
				}
				if token == "" {
					fmt.Println("Token is required")
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Println("Invalid url scheme")
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				if u.Scheme == "aio" {
					serverAddr = "127.0.0.1:8448"
				} else if u.Scheme == "iad" {
					serverAddr = "api.ea.iad3.rackfs.com:8443"
				} else {
					fmt.Printf("Invalid region %s", u.Scheme)
					os.Exit(1)
				}
				acctNum = u.Host
				fsNum = u.Path[1:]
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.DeleteFS(context.Background(), &mb.DeleteFSRequest{Acctnum: acctNum, FSid: fsNum, Token: token})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
				log.Printf("Delete Results: %s", result.Payload)
			},
		},
		{
			Name:  "update",
			Usage: "Update a File Systems",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, N",
					Value: "",
					Usage: "Name of the file system",
				},
				cli.StringFlag{
					Name:  "S, status",
					Value: "",
					Usage: "Status of the file system",
				},
			},
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for delete.")
					os.Exit(1)
				}
				if token == "" {
					fmt.Println("Token is required")
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Printf("Url Parse error: %v", err)
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				if u.Scheme == "aio" {
					serverAddr = "127.0.0.1:8448"
				} else if u.Scheme == "iad" {
					serverAddr = "api.ea.iad3.rackfs.com:8443"
				} else {
					fmt.Printf("Invalid region %s", u.Scheme)
					os.Exit(1)
				}
				acctNum = u.Host
				fsNum = u.Path[1:]
				if c.String("name") != "" && !validAcctName(c.String("name")) {
					fmt.Printf("Invalid File System String: %q\n", c.String("name"))
					os.Exit(1)
				}
				fsMod := &mb.ModFS{
					Name:   c.String("name"),
					Status: c.String("status"),
				}
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.UpdateFS(context.Background(), &mb.UpdateFSRequest{Acctnum: acctNum, FSid: fsNum, Token: token, Filesys: fsMod})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
				log.Printf("Update Results: %s", result.Payload)
			},
		},
		{
			Name:  "grant",
			Usage: "Grant an Addr access to a File Systems",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "addr",
					Value: "",
					Usage: "Address to Grant",
				},
			},
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for delete.")
					os.Exit(1)
				}
				if token == "" {
					fmt.Println("Token is required")
					os.Exit(1)
				}
				if c.String("addr") == "" {
					fmt.Println("addr is required")
					os.Exit(1)
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Println("Invalid url scheme")
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				if u.Scheme == "aio" {
					serverAddr = "127.0.0.1:8448"
				} else if u.Scheme == "iad" {
					serverAddr = "api.ea.iad3.rackfs.com:8443"
				} else {
					fmt.Printf("Invalid region %s", u.Scheme)
					os.Exit(1)
				}
				acctNum = u.Host
				fsNum = u.Path[1:]
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.GrantAddrFS(context.Background(), &mb.GrantAddrFSRequest{Acctnum: acctNum, FSid: fsNum, Token: token, Addr: c.String("addr")})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
			},
		},
		{
			Name:  "revoke",
			Usage: "Revoke an Addr's access to a File Systems",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "addr",
					Value: "",
					Usage: "Address to Revoke",
				},
			},
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for revoke.")
					os.Exit(1)
				}
				if token == "" {
					fmt.Println("Token is required")
					os.Exit(1)
				}
				if c.String("addr") == "" {
					fmt.Println("addr is required")
					os.Exit(1)
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Println("Invalid url scheme")
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				if u.Scheme == "aio" {
					serverAddr = "127.0.0.1:8448"
				} else if u.Scheme == "iad" {
					serverAddr = "api.ea.iad3.rackfs.com:8443"
				} else {
					fmt.Printf("Invalid region %s", u.Scheme)
					os.Exit(1)
				}
				acctNum = u.Host
				fsNum = u.Path[1:]
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.RevokeAddrFS(context.Background(), &mb.RevokeAddrFSRequest{Acctnum: acctNum, FSid: fsNum, Token: token, Addr: c.String("addr")})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
			},
		},
		{
			Name:  "verify",
			Usage: "Verify an Addr has access to a file system",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "addr",
					Value: "",
					Usage: "Address to check",
				},
			},
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for revoke.")
					os.Exit(1)
				}
				if c.String("addr") == "" {
					fmt.Println("addr is required")
					os.Exit(1)
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Println("Invalid url scheme")
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				fsNum = u.Host
				conn := setupWS(serverAddr)
				ws := mb.NewFileSystemAPIClient(conn)
				result, err := ws.LookupAddrFS(context.Background(), &mb.LookupAddrFSRequest{FSid: fsNum, Addr: c.String("addr")})
				if err != nil {
					log.Fatalf("Bad Request: %v", err)
					conn.Close()
					os.Exit(1)
				}
				conn.Close()
				log.Printf("Result: %s\n", result.Status)
			},
		},
		{
			Name:  "mount",
			Usage: "mount a file system",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "o",
					Value: "",
					Usage: "mount options",
				},
			},
			Action: func(c *cli.Context) {
				if !c.Args().Present() {
					fmt.Println("Invalid syntax for revoke.")
					os.Exit(1)
				}
				if c.String("o") == "" {
					fmt.Println("options are required")
					os.Exit(1)
				}
				u, err := url.Parse(c.Args().Get(0))
				if err != nil {
					fmt.Println("Invalid url scheme")
					os.Exit(1)
				}
				fmt.Println(u.Scheme)
				if u.Scheme == "aio" {
					serverAddr = "127.0.0.1:8448"
				} else if u.Scheme == "iad" {
					serverAddr = "api.ea.iad3.rackfs.com:8443"
				} else {
					fmt.Printf("Invalid region %s", u.Scheme)
					os.Exit(1)
				}
				if u.Host == "" {
					fmt.Printf("File System id is required")
					os.Exit(1)
				}
				fsnum := u.Host
				mountpoint := c.Args().Get(1)
				// check mountpoint exists
				if _, ferr := os.Stat(mountpoint); os.IsNotExist(ferr) {
					log.Printf("Mount point %s does not exist\n\n", mountpoint)
					os.Exit(1)
				}
				fmt.Println("run fusermountPath")
				fusermountPath()
				// process file system options
				clargs := getArgs(c.String("o"))
				fmt.Println(clargs)
				// crapy debug log handling :)
				if debug, ok := clargs["debug"]; ok {
					if debug == "false" {
						log.SetFlags(0)
						log.SetOutput(ioutil.Discard)
					}
				} else {
					log.SetFlags(0)
					log.SetOutput(ioutil.Discard)
				}
				// Setup grpc
				fmt.Println("Setting up grpc")
				var opts []grpc.DialOption
				creds := credentials.NewTLS(&tls.Config{
					InsecureSkipVerify: true,
				})
				opts = append(opts, grpc.WithTransportCredentials(creds))
				conn, err := grpc.Dial(serverAddr, opts...)
				if err != nil {
					log.Fatalf("failed to dial: %v", err)
				}
				defer conn.Close()
				// Work with fuse
				fmt.Println("Work with fuse")
				cfs, err := fuse.Mount(
					mountpoint,
					fuse.FSName("cfs"),
					fuse.Subtype("cfs"),
					fuse.LocalVolume(),
					fuse.VolumeName("CFS"),
					//fuse.AllowOther(),
					fuse.DefaultPermissions(),
				)
				if err != nil {
					log.Fatal(err)
				}
				defer cfs.Close()

				rpc := newrpc(conn)
				fs := newfs(cfs, rpc)
				srv := newserver(fs)

				// Verify fsnum and ip
				// 1. Get local IP Address
				// 2. Check for valid filesystem
				// 		query group store
				//			"/fs"    "[fsnum]"
				// 3. Check for valid ip
				//		query group store
				//			"/fs/[fsnum]/addr"		"[ipaddress]"

				fmt.Printf("Verify that file system %s with ip %s ", fsnum, "127.0.0.1")

				if err := srv.serve(); err != nil {
					log.Fatal(err)
				}

				<-cfs.Ready
				if err := cfs.MountError; err != nil {
					log.Fatal(err)
				}
			},
		},
	}
	app.Run(os.Args)
}

// getArgs is passed a command line and breaks it up into commands
// the valid format is <device> <mount point> -o [Options]
func getArgs(args string) map[string]string {
	// Setup declarations
	var optList []string
	requiredOptions := []string{}
	clargs := make(map[string]string)

	// process options -o
	optList = strings.Split(args, ",")
	for _, item := range optList {
		if strings.Contains(item, "=") {
			value := strings.Split(item, "=")
			if value[0] == "" || value[1] == "" {
				log.Printf("Invalid option %s, %s no value\n\n", value[0], value[1])
				os.Exit(1)
			} else {
				clargs[value[0]] = value[1]
			}
		} else {
			clargs[item] = ""
		}
	}

	// Verify required options exist
	for _, v := range requiredOptions {
		_, ok := clargs[v]
		if !ok {
			log.Printf("%s is a required option", v)
			os.Exit(1)
		}
	}

	// load in device and mountPoint
	return clargs
}

func fusermountPath() {
	// Grab the current path
	currentPath := os.Getenv("PATH")
	if len(currentPath) == 0 {
		// using mount seem to not have a path
		// fusermount is in /bin
		os.Setenv("PATH", "/bin")
	}
}

// Validate the account string passed in from the command line
func validAcctName(a string) bool {
	//TODO: Determine what needs to be done to validate
	return true
}

// setupWS ...
func setupWS(svr string) *grpc.ClientConn {
	var opts []grpc.DialOption
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	opts = append(opts, grpc.WithTransportCredentials(creds))
	conn, err := grpc.Dial(svr, opts...)
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	return conn
}
