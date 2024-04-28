package client

import (
	"fmt"
	"time"

	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/integration/cmn"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/integration/nwo"
	"github.com/hyperledger/fabric/integration/nwo/commands"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func AddUser(network *nwo.Network, peer *nwo.Peer, orderer *nwo.Orderer, user *UserFoundation) {
	sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeInvoke{
		ChannelID: cmn.ChannelAcl,
		Orderer:   network.OrdererAddress(orderer, nwo.ListenPort),
		Name:      cmn.ChannelAcl,
		Ctor:      fmt.Sprintf(`{"Args":["addUser", "%s", "test", "testuser", "true"]}`, user.PublicKeyBase58),
		PeerAddresses: []string{
			network.PeerAddress(network.Peer("Org1", "peer0"), nwo.ListenPort),
			network.PeerAddress(network.Peer("Org2", "peer0"), nwo.ListenPort),
		},
		WaitForEvent: true,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit(0))
	Expect(sess.Err).To(gbytes.Say("Chaincode invoke successful. result: status:200"))

	CheckUser(network, peer, user)
}

func CheckUser(network *nwo.Network, peer *nwo.Peer, user *UserFoundation) {
	Eventually(func() string {
		sess, err := network.PeerUserSession(peer, "User1", commands.ChaincodeQuery{
			ChannelID: cmn.ChannelAcl,
			Name:      cmn.ChannelAcl,
			Ctor:      fmt.Sprintf(`{"Args":["checkKeys", "%s"]}`, user.PublicKeyBase58),
		})
		Eventually(sess, network.EventuallyTimeout).Should(gexec.Exit())
		if sess.ExitCode() != 0 {
			return fmt.Sprintf("exit code is %d: %s, %v", sess.ExitCode(), string(sess.Err.Contents()), err)
		}

		out := sess.Out.Contents()[:len(sess.Out.Contents())-1] // skip line feed
		resp := &pb.AclResponse{}
		err = proto.Unmarshal(out, resp)
		if err != nil {
			return fmt.Sprintf("failed to unmarshal response: %v", err)
		}

		addr := base58.CheckEncode(resp.GetAddress().GetAddress().GetAddress()[1:], resp.GetAddress().GetAddress().GetAddress()[0])
		if addr != user.AddressBase58Check {
			return fmt.Sprintf("Error: expected %s, received %s", user.AddressBase58Check, addr)
		}

		return ""
	}, network.EventuallyTimeout, time.Second).Should(BeEmpty())
}
