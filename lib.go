package operation_util

import (
	"archive/tar"
	"bufio"
	"fmt"
	"github.com/gobuffalo/packr"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
)

type ContextManager struct {
	context string
}


var (
	contextManager ContextManager
	sshPasswd      string
)

type Connection struct {
	SshClient *ssh.Client
	SftpClient *sftp.Client
}

/**
change directory in local host
 */
func Lcd(dir string){
	if err := os.Chdir(dir); err != nil{
		panic(err)
	}

}

/**
run command in local host
 */
func Local(name string, arg ...string){
	command := exec.Command(name, arg...)
	stdout, err := command.StdoutPipe()
	if err != nil{
		panic(err)
	}

	if err := command.Start(); err != nil{
		panic("An issue occured: "+err.Error())
		//fmt.Println("An issue occured: ", err)
		return
	}

	localBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		panic("ReadAll Stdout: "+err.Error())
		//fmt.Println("ReadAll Stdout:", err.Error())
		return
	}

	fmt.Println(string(localBytes))
}

func sshDial(remoteAddr string, config ssh.ClientConfig) Connection {
	sshClient, err := ssh.Dial("tcp", remoteAddr, &config)
	if err != nil {
		panic(err)
	}else{
		fmt.Println("Connected to manipulator host")
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil{
		fmt.Println("Failed to create sftp client")
		panic(err)
	}
	return Connection{
		sshClient,
		sftpClient,

	}
}

/**
connect to manipulator host via SSH
 */
func SSHConnect(user string, password string, remoteAddr string) Connection {
	sshPasswd = password
	PassWd := []ssh.AuthMethod{ssh.Password(password)}
	config := ssh.ClientConfig{User: user, Auth: PassWd, HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}}

	return sshDial(remoteAddr, config)
}

func SSHConnectWithPrivateKey(user string, remoteAddr string) Connection {
	keyPassPhrase := "kcswag"
	sshPasswd = keyPassPhrase
	keyBox := packr.NewBox("./jry-key")
	pemByte := keyBox.Bytes("kcswag.pem")
	signer, err := ssh.ParsePrivateKeyWithPassphrase(pemByte, []byte(keyPassPhrase))
	if err != nil{
		fmt.Println("Failed to parse private key")
		panic(err)
	}
	authMethods := []ssh.AuthMethod{ssh.PublicKeys(signer)}
	config := ssh.ClientConfig{
		User:user,
		Auth:authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return sshDial(remoteAddr, config)

}

/**
change directory in manipulator host
 */
func Cd(dir string){
	contextManager.context += "cd " + dir + " && "
}


/**
run command in manipulator host
 */
func (conn *Connection) Run(cmd string){

	sshSession, err := conn.SshClient.NewSession()
	defer sshSession.Close()
	if err != nil{
		panic(err)
	}

	runByte, runErr := sshSession.Output(contextManager.context + cmd)
	fmt.Println("Running command...")
	if runErr != nil{
		panic("An exception occured while executing command: "+runErr.Error())
	}
	fmt.Println(string(runByte))
}

func (conn *Connection) Sudo(cmd string){
	sshSession, err := conn.SshClient.NewSession()
	defer sshSession.Close()
	if err != nil{
		panic(err)
	}

	sudoPrefix := "echo '" + sshPasswd + "' | sudo -S "
	cmd = sudoPrefix+cmd
	runByte, runErr := sshSession.Output(contextManager.context + cmd)
	fmt.Println("Running command...")
	if runErr != nil{
		panic("An exception occured while executing command: "+runErr.Error())
	}
	fmt.Println(string(runByte))
}

/**
get cli input in interactive way
 */
func Prompt(display string, dft string) string{
	if dft != ""{
		print(display+" ["+dft+"]"+": ")
	}else{
		dft = "update"
		print(display+": ")
	}
	bio := bufio.NewReader(os.Stdin)
	line, _, err := bio.ReadLine()
	if err != nil{
		panic(err)
	}
	if line == nil && dft != ""{
		return dft
	}else{
		return string(line)
	}

}

func (conn *Connection) Put(localPath, remotePath string){
	Upload(conn.SftpClient, localPath, remotePath)
}

func ReadFile(path string) string{
	content, err := ioutil.ReadFile(path)
	if err != nil{
		panic(err)
	}
	return string(content)
}

func WriteFile(filename string, content string){
	if err := ioutil.WriteFile(filename,[]byte(content),755); err != nil{
		panic(err)
	}

}

func CopyFile(srcFileName, destFileName string){
	srcFile ,err := os.Open(srcFileName)
	if err != nil{
		fmt.Println(err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destFileName,os.O_RDWR|os.O_CREATE,0755)
	if err != nil{
		fmt.Println(err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile,srcFile)
	if err != nil{
		panic(err)
	}

}

func TarCompress(srcFilePath, targetFilePath string ){
	targetFile ,err := os.Create(targetFilePath)
	if err != nil{
		panic(err)
	}
	defer targetFile.Close()

	tarWriter := tar.NewWriter(targetFile)
	defer tarWriter.Close()

	//get file header
	fileInfo, err := os.Stat(srcFilePath)
	if err != nil{
		panic(err)
	}

	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil{
		panic(err)
	}

	//write header
	err = tarWriter.WriteHeader(header)
	if err != nil{
		panic(err)
	}

	srcFile, err := os.Open(srcFilePath)
	if err != nil{
		panic(err)
	}

	//write src file to tar file
	io.Copy(tarWriter, srcFile)

}

func IsDirExistOrMake(path string) error{
	var isDirExist bool
	_, err := os.Stat(path)
	if err == nil{
		isDirExist = true
	}
	if os.IsNotExist(err){
		isDirExist = false
	}

	if !isDirExist {
		mkErr := os.Mkdir(path, 0755)
		if mkErr != nil{
			return mkErr
		}
	}

	return nil
}


func IsExistOrCreate(filePath string) *os.File{
	file,err := os.OpenFile(filePath, os.O_RDWR,0)
	if err!=nil && os.IsNotExist(err){
		file,_ = os.Create(filePath)
		return file
	}
	return file
}

func IfLocalPathFileExist() *os.File{
	filename := "local-path"
	return IsExistOrCreate(filename)
}


func GetLocalPath(localPathFile *os.File) string{
	readPath := ReadFile(localPathFile.Name())
	var localPath string
	if readPath == ""{
		localPath = Prompt("Please input absolute local path", "")
	}else{
		localPath = Prompt("Please input absolute local path",readPath)
	}

	if localPath == "" && readPath == ""{
		return GetLocalPath(localPathFile)
	}else if localPath != ""{
		lastLetter := localPath[len(localPath)-1: len(localPath)]
		if lastLetter != "/"{
			localPath += "/"
		}

		_,err :=os.Stat(localPath)
		if err != nil{
			if os.IsNotExist(err) {
				fmt.Println("File provided does not exist! Please try again! ")
				return GetLocalPath(localPathFile)
			}else{
				fmt.Println("Invalid path provided! Please try again! ")
			}
		}

		pathErr := os.Chdir(localPath)
		if pathErr != nil{
			fmt.Println(pathErr)
			fmt.Println("Invalid directory, please try again!")
			return GetLocalPath(localPathFile)
		}else{
			fmt.Println(localPath)
			_, err := localPathFile.Write([]byte(localPath))
			if err != nil{
				fmt.Println(err)
				return GetLocalPath(localPathFile)
			}
		}

	}
	return localPath
}

func setEnv(cgoEnabled,goos,goarch string){
	os.Setenv("CGO_ENABLED","0")
	os.Setenv("GOOS","linux")
	os.Setenv("GOARCH","amd64")
}

func GoBuild(dir string){
	setEnv("0","linux","amd64")
	Lcd(dir)
	Local("go", "build")
}

func PackrBuild(dir string,cgoEnabled bool,goos, goarch string){
	var cgo string
	if cgoEnabled{
		cgo = "1"
	}else{
		cgo = "0"
	}
	setEnv(cgo,goos,goarch)
	if dir != ""{
		Lcd(dir)
	}
	Local("packr","build")
}

func GitPush(localPath string){
	commit := Prompt("Please input commit","update")
	Lcd(localPath)
	Local("git","add",".")
	mPhase := fmt.Sprintf("-m '%s'",commit)
	Local("git","commit",mPhase,)
}

