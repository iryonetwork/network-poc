package main

import (
	"context"
	"fmt"

	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eth"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/specs"
	"google.golang.org/grpc/metadata"
)

const tokenKey = "token"

type rpcServer struct {
	config  *config.Config
	keySent map[string]chan specs.Event_KeySentDetails
	eth     *eth.Storage
	ehr     *ehr.Storage
}

func (s *rpcServer) Login(ctx context.Context, request *specs.LoginRequest) (*specs.LoginResponse, error) {
	// mock login
	s.config.Tokens["token"] = request.Public
	return &specs.LoginResponse{
		Token: "token",
	}, nil
}

func (s *rpcServer) Upload(ctx context.Context, request *specs.UploadRequest) (*specs.Empty, error) {
	user, err := s.loggedIn(ctx)
	if err != nil {
		return nil, err
	}

	granted, err := s.eth.AccessGranted(request.Owner, user)
	if err != nil {
		return nil, err
	}

	if !granted {
		return nil, fmt.Errorf("You do not have permission to upload this file")
	}

	s.ehr.Save(request.Owner, request.Data)

	return &specs.Empty{}, nil
}

func (s *rpcServer) Download(ctx context.Context, request *specs.DownloadRequest) (*specs.DownloadResponse, error) {
	user, err := s.loggedIn(ctx)
	if err != nil {
		return nil, err
	}

	granted, err := s.eth.AccessGranted(request.Owner, user)
	if err != nil {
		return nil, err
	}

	if !granted {
		return nil, fmt.Errorf("You do not have permission to download this file")
	}

	document := s.ehr.Get(request.Owner)
	if document == nil {
		return nil, fmt.Errorf("Document for %s does not exist", request.Owner)
	}

	return &specs.DownloadResponse{
		Data: document,
	}, nil
}

func (s *rpcServer) SendKey(ctx context.Context, request *specs.SendKeyRequest) (*specs.Empty, error) {
	user, err := s.loggedIn(ctx)
	if err != nil {
		return nil, err
	}

	granted, err := s.eth.AccessGranted(user, request.To)
	if err != nil {
		return nil, err
	}

	if !granted {
		return nil, fmt.Errorf("User %s does not have permission to receive key from %s", request.To, user)
	}

	if keys, ok := s.keySent[request.To]; ok {
		keys <- specs.Event_KeySentDetails{
			From: user,
			Key:  request.Key,
		}
		return &specs.Empty{}, nil
	}

	return nil, fmt.Errorf("Receiver for %s not registerd", request.To)
}

func (s *rpcServer) Subscribe(_ *specs.Empty, stream specs.Cloud_SubscribeServer) error {
	user, err := s.loggedIn(stream.Context())
	if err != nil {
		return err
	}

	if _, ok := s.keySent[user]; !ok {
		s.keySent[user] = make(chan specs.Event_KeySentDetails)
	}

	for {
		key := <-s.keySent[user]
		err := stream.Send(&specs.Event{
			Type:           specs.Event_KeySent,
			KeySentDetails: &key,
		})

		if err != nil {
			return err
		}
	}
}

func (s *rpcServer) loggedIn(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("Error getting metadata")
	}

	if token, ok := md[tokenKey]; ok && len(token) > 0 {
		if user, ok := s.config.Tokens[token[0]]; ok {
			return user, nil
		}
	}
	return "", fmt.Errorf("Not logged in")
}
