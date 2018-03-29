package model

import (
	"fmt"
	webshell "git.trapti.tech/CPCTF2018/webshell/rpc"
	"google.golang.org/grpc"
	"os"
)

var webShellConn *grpc.ClientConn
var webShellCli *webshell.WebShellClient

//InitWebShellCli Initialize Web Shell Client
func InitWebShellCli() error {
	conn, err := grpc.Dial(os.Getenv("WEBSHELL_GRPC_TARGET"), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect webshell grpc: %v", err)
	}
	webShellConn = conn
	cli := webshell.NewWebShellClient(conn)
	webShellCli = &cli
	return nil
}

//TermWebShellCli Terminate Web Shell Client
func TermWebShellCli() {
	webShellConn.Close()
}