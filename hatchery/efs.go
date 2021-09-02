package hatchery

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/efs"
)

type EFS struct {
	EFSArn       string
	FileSystemId string
}

func (creds *CREDS) getEFSFileSystem(username string, svc *efs.EFS) (*efs.DescribeFileSystemsOutput, error) {
	input := &efs.DescribeFileSystemsInput{
		CreationToken: aws.String(userToResourceName(username, "pod")),
	}
	result, err := svc.DescribeFileSystems(input)
	if err != nil {
		return nil, fmt.Errorf("Failed to describe EFS FS: %s", err)
	}
	// return empty struct if no filesystems are found
	if len(result.FileSystems) == 0 {
		return nil, nil
	}
	return result, nil
}

func (creds *CREDS) createMountTarget(FileSystemId string, svc *efs.EFS) (*efs.MountTargetDescription, error) {
	_, subnets, securityGroups, err := creds.describeDefaultNetwork()

	input := &efs.CreateMountTargetInput{
		FileSystemId: aws.String(FileSystemId),
		SubnetId:     subnets.Subnets[0].SubnetId,
		// TODO: Make this correct, currently it's all using the same SG
		SecurityGroups: []*string{
			securityGroups.SecurityGroups[0].GroupId,
		},
	}

	result, err := svc.CreateMountTarget(input)
	if err != nil {
		return nil, fmt.Errorf("Failed to create mount target: %s", err)
	}
	return result, nil
}

func (creds *CREDS) createAccessPoint(FileSystemId string, username string, svc *efs.EFS) (*efs.CreateAccessPointOutput, error) {

	input := &efs.CreateAccessPointInput{
		ClientToken:  aws.String(fmt.Sprintf("ap-%s", userToResourceName(username, "pod"))),
		FileSystemId: aws.String(FileSystemId),
		PosixUser: &efs.PosixUser{
			Gid: aws.Int64(100),
			Uid: aws.Int64(1000),
		},
		RootDirectory: &efs.RootDirectory{
			CreationInfo: &efs.CreationInfo{
				OwnerGid:    aws.Int64(100),
				OwnerUid:    aws.Int64(1000),
				Permissions: aws.String("0755"),
			},
			Path: aws.String("/"),
		},
	}

	result, err := svc.CreateAccessPoint(input)
	if err != nil {
		return nil, fmt.Errorf("Failed to create accessPoint: %s", err)
	}
	return result, nil
}

func (creds *CREDS) EFSFileSystem(username string) (*EFS, error) {
	svc := efs.New(session.New(&aws.Config{
		Credentials: creds.creds,
		// TODO: Make this configurable
		Region: aws.String("us-east-1"),
	}))
	exisitingFS, _ := creds.getEFSFileSystem(username, svc)
	Config.Logger.Printf("Existiing FS: %s", exisitingFS)
	if exisitingFS == nil {
		input := &efs.CreateFileSystemInput{
			Backup:          aws.Bool(false),
			CreationToken:   aws.String(userToResourceName(username, "pod")),
			Encrypted:       aws.Bool(true),
			PerformanceMode: aws.String("generalPurpose"),
			Tags: []*efs.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(userToResourceName(username, "pod")),
				},
			},
		}

		result, err := svc.CreateFileSystem(input)
		if err != nil {
			return nil, fmt.Errorf("Error creating EFS filesystem: %s", err)
		}

		exisitingFS, _ = creds.getEFSFileSystem(username, svc)
		for *exisitingFS.FileSystems[0].LifeCycleState != "available" {
			Config.Logger.Printf("EFS filesystem is in state: %s ...  Waiting for 2 seconds", *exisitingFS.FileSystems[0].LifeCycleState)
			// sleep for 2 sec
			time.Sleep(1 * time.Second)
			exisitingFS, _ = creds.getEFSFileSystem(username, svc)
		}

		// Create mount target
		mountTarget, err := creds.createMountTarget(*result.FileSystemId, svc)
		if err != nil {
			return nil, fmt.Errorf("Failed to create EFS MountTarget: %s", err)
		}
		Config.Logger.Printf("MountTarget created: %s", *mountTarget.MountTargetId)
		accessPoint, err := creds.createAccessPoint(*result.FileSystemId, username, svc)
		if err != nil {
			return nil, fmt.Errorf("Failed to create EFS AccessPoint: %s", err)
		}
		Config.Logger.Printf("AccessPoint created: %s", *accessPoint.AccessPointId)

		return &EFS{
			EFSArn:       *result.FileSystemArn,
			FileSystemId: *result.FileSystemId,
		}, nil
	} else {
		return &EFS{
			EFSArn:       *exisitingFS.FileSystems[0].FileSystemArn,
			FileSystemId: *exisitingFS.FileSystems[0].FileSystemId,
		}, nil
	}
}
