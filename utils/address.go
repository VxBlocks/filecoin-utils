package utils

import (
	"context"
	"encoding/json"
	
	"github.com/urfave/cli/v2"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"golang.org/x/xerrors"

)

type EXAddressDescription struct {
	ID       string
	Filecoin address.Address
	Eth      ethtypes.EthAddress
	Type     string
}


var ExAddressTransformationCmd = &cli.Command{
	Name:      "addr-description",
	Aliases:   []string{"addrdescription"},
	Usage:     "Get ID Fil Eth address from id/fil/eth address",
	ArgsUsage: "address",
	Action: func(cctx *cli.Context) error {
		if argc := cctx.Args().Len(); argc < 1 {
			return xerrors.Errorf("must pass the address(id/fil/eth)")
		}

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := ReqContext(cctx)

		addrString := cctx.Args().Get(0)

		var out EXAddressDescription

		var faddr address.Address
		var eaddr ethtypes.EthAddress
		addr, err := address.NewFromString(addrString)
		if err != nil { // This isn't a filecoin address
			eaddr, err = ethtypes.ParseEthAddress(addrString)
			if err != nil { // This isn't an Eth address either
				return xerrors.Errorf("address is not a filecoin or eth address")
			}
			faddr, err = eaddr.ToFilecoinAddress()
			if err != nil {
				return err
			}
		} else {
			eaddr, faddr, err = ethAddrFromFilecoinAddress(ctx, addr, api)
			if err != nil {
				return err
			}
		}

		newfaddr, err := api.StateAccountKey(ctx, faddr, types.EmptyTSK)
		if err == nil {
			faddr = newfaddr
		}

		out.Filecoin = faddr
		out.Eth = eaddr

		actor, err := api.StateGetActor(ctx, faddr, types.EmptyTSK)
		if err == nil {
			id, err := api.StateLookupID(ctx, faddr, types.EmptyTSK)
			if err != nil {
				out.ID = "n/a"
			} else {
				out.ID = id.String()
			}
			if name, _, ok := actors.GetActorMetaByCode(actor.Code); ok {
				out.Type = name
			} else {
				out.Type = "unknown"
			}
		} else {
			out.ID = "unknown"
			out.Type = "unknown"
		}

		byte, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		afmt := NewAppFmt(cctx.App)
		afmt.Println(string(byte))
		return nil
	},
}

func ethAddrFromFilecoinAddress(ctx context.Context, addr address.Address, fnapi v0api.FullNode) (ethtypes.EthAddress, address.Address, error) {
	var faddr address.Address
	var err error

	switch addr.Protocol() {
	case address.BLS, address.SECP256K1:
		faddr, err = fnapi.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return ethtypes.EthAddress{}, addr, err
		}
	case address.Actor, address.ID:
		faddr, err = fnapi.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			return ethtypes.EthAddress{}, addr, err
		}
		fAct, err := fnapi.StateGetActor(ctx, faddr, types.EmptyTSK)
		if err != nil {
			return ethtypes.EthAddress{}, addr, err
		}
		if fAct.DelegatedAddress != nil && (*fAct.DelegatedAddress).Protocol() == address.Delegated {
			faddr = *fAct.DelegatedAddress
		}
	case address.Delegated:
		faddr = addr
	default:
		return ethtypes.EthAddress{}, addr, xerrors.Errorf("Filecoin address doesn't match known protocols")
	}

	ethAddr, err := ethtypes.EthAddressFromFilecoinAddress(faddr)
	if err != nil {
		return ethtypes.EthAddress{}, addr, err
	}

	return ethAddr, faddr, nil
}