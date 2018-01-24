package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/iryonetwork/network-poc/logger"
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
	log     *logger.Log
}

func (s *rpcServer) Login(ctx context.Context, request *specs.LoginRequest) (*specs.LoginResponse, error) {
	s.log.Debugf("RPCServer::Login(%+v) called", request)

	// verify signature
	pub, err := crypto.DecompressPubkey(request.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress public key; %v", err)
	}

	signature := struct {
		R, S *big.Int
	}{}

	_, err = asn1.Unmarshal(request.Signature, &signature)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal signature; %v", err)
	}

	if !ecdsa.Verify(pub, request.Hash, signature.R, signature.S) {
		return nil, fmt.Errorf("invalid login signature")
	}

	// save login token
	token := hex.EncodeToString(request.Signature)
	s.config.Tokens[token] = crypto.PubkeyToAddress(*pub).String()
	return &specs.LoginResponse{
		Token:           token,
		ContractAddress: s.config.EthContractAddr,
	}, nil
}

func (s *rpcServer) Upload(ctx context.Context, request *specs.UploadRequest) (*specs.Empty, error) {
	s.log.Debugf("RPCServer::Upload(%+v) called", request)

	//check permissions
	user, err := s.loggedIn(ctx)
	if err != nil {
		return nil, err
	}

	granted, err := s.eth.AccessGranted(request.Owner, user)
	if err != nil {
		return nil, fmt.Errorf("failed to check if access is granted; %v", err)
	}

	if !granted {
		return nil, fmt.Errorf("You do not have permission to upload this file")
	}

	// save file to storage
	s.ehr.Save(request.Owner, request.Data)

	return &specs.Empty{}, nil
}

func (s *rpcServer) Download(ctx context.Context, request *specs.DownloadRequest) (*specs.DownloadResponse, error) {
	s.log.Debugf("RPCServer::Download(%+v) called", request)

	// check permissions
	user, err := s.loggedIn(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not logged in; %v", err)
	}

	granted, err := s.eth.AccessGranted(request.Owner, user)
	if err != nil {
		return nil, fmt.Errorf("failed to check accessGranted; %v", err)
	}

	if !granted {
		return nil, fmt.Errorf("You do not have permission to download this file")
	}

	// get document from storage
	document := s.ehr.Get(request.Owner)
	if document == nil {
		return nil, fmt.Errorf("Document for %s does not exist", request.Owner)
	}

	// return document
	return &specs.DownloadResponse{
		Data: document,
	}, nil
}

func (s *rpcServer) SendKey(ctx context.Context, request *specs.SendKeyRequest) (*specs.Empty, error) {
	s.log.Debugf("RPCServer::SendKey(%+v) called", request)

	// check permissions
	user, err := s.loggedIn(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not logged in; %v", err)
	}

	granted, err := s.eth.AccessGranted(user, request.To)
	if err != nil {
		return nil, fmt.Errorf("failed to check AccessGranted; %v", err)
	}

	if !granted {
		return nil, fmt.Errorf("User %s does not have permission to receive key from %s", request.To, user)
	}

	// send key to users's channel
	if keys, ok := s.keySent[request.To]; ok {
		keys <- specs.Event_KeySentDetails{
			From: user,
			Key:  request.Key,
		}
		return &specs.Empty{}, nil
	}

	return nil, fmt.Errorf("Receiver for %s not registered", request.To)
}

func (s *rpcServer) Subscribe(_ *specs.Empty, stream specs.Cloud_SubscribeServer) error {
	s.log.Debugf("RPCServer::Subscribe() called")

	// check permissions
	user, err := s.loggedIn(stream.Context())
	if err != nil {
		return fmt.Errorf("user not logged in; %v", err)
	}

	if _, ok := s.keySent[user]; !ok {
		s.keySent[user] = make(chan specs.Event_KeySentDetails)
	}

	for {
		// send key from channel to user
		key := <-s.keySent[user]

		s.log.Debugf("received a new keySent event (%+v)", key)

		err := stream.Send(&specs.Event{
			Type:           specs.Event_KeySent,
			KeySentDetails: &key,
		})

		if err != nil {
			return fmt.Errorf("failed to send event to stream; %v", err)
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
