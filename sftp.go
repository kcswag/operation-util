package operation_util

import (
	"fmt"
	"github.com/pkg/sftp"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func uploadFile(sftpClient *sftp.Client, localPath, remotePath string){
	dstFile, err := sftpClient.Create(remotePath)
	if err != nil{
		fmt.Println("sftpClient.Create error: ", remotePath)
		panic(err)
	}
	dstFile.Close()

	srcFileContent, err := ioutil.ReadFile(localPath)
	if err != nil{
		fmt.Println("read file error: ",localPath)
		panic(err)
	}

	dstFile.Write(srcFileContent)
	fmt.Println("File is copied to manipulator server successfully")
}

func Upload(sftpClient *sftp.Client, localDir, remoteDir string){
	localF, _ := os.Stat(localDir)
	if localF.IsDir() {
		localFiles, err := ioutil.ReadDir(localDir)
		if err != nil{
			log.Println("Failed to read directory: ",err)
		}

		for _, file := range localFiles{
			localFilePath := path.Join(localDir, file.Name())
			remoteFilePath := path.Join(remoteDir,file.Name())
			if file.IsDir(){
				sftpClient.Mkdir(remoteDir)
				Upload(sftpClient, localFilePath, remoteFilePath)
			}else{
				uploadFile(sftpClient, localFilePath,remoteFilePath)
			}
		}
	}else{
		uploadFile(sftpClient, localDir, remoteDir)
	}

	fmt.Println("Successfully copy directory/file to manipulator host!")
}
