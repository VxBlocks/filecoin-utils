package utils

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin/v10/eam"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

type EXSignedMessage struct {
	types.SignedMessage
	MethodName   string
	ParamJson    string
	ReturnJson   string
	Events       []EXEvent
	GasUsed      int64
	BaseFee      int64
	Status       string
	ErrorMessage string
}

type EXPerMessage struct {
	CID          cid.Cid
	MethodName   string
	ReturnJson   string
	Events       []EXEvent
	GasUsed      int64
	Status       string
	ErrorMessage string
}

type smEXSignedMessage struct {
	*types.RawSignedMessage
	CID          cid.Cid
	MethodName   string
	ParamJson    string
	ReturnJson   string
	Events       []EXEvent
	GasUsed      int64
	BaseFee      int64
	Status       string
	ErrorMessage string
}

type EXMessage struct {
	types.Message
	MethodName   string
	ParamJson    string
	ReturnJson   string
	Events       []EXEvent
	GasUsed      int64
	BaseFee      int64
	Status       string
	ErrorMessage string
}

type smEXMessageCid struct {
	*types.RawMessage
	CID          cid.Cid
	MethodName   string
	ParamJson    string
	ReturnJson   string
	Events       []EXEvent
	GasUsed      int64
	BaseFee      int64
	Status       string
	ErrorMessage string
}

type EXEvent struct {
	Address address.Address
	Topics  []EXEventEntry
}

type EXEventEntry struct {
	Flags uint8
	Key   string
	Codec uint64
	Value string
}

type ExCreateExternalReturn struct {
	ActorID       uint64
	RobustAddress *address.Address
	EthAddress    string
}

func (sm *EXMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&smEXMessageCid{
		RawMessage:   (*types.RawMessage)(&sm.Message),
		CID:          sm.Cid(),
		MethodName:   sm.MethodName,
		ParamJson:    sm.ParamJson,
		ReturnJson:   sm.ReturnJson,
		Events:       sm.Events,
		GasUsed:      sm.GasUsed,
		BaseFee:      sm.BaseFee,
		Status:       sm.Status,
		ErrorMessage: sm.ErrorMessage,
	})
}

func (sm *EXSignedMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(&smEXSignedMessage{
		RawSignedMessage: (*types.RawSignedMessage)(&sm.SignedMessage),
		CID:              sm.Cid(),
		MethodName:       sm.MethodName,
		ParamJson:        sm.ParamJson,
		ReturnJson:       sm.ReturnJson,
		Events:           sm.Events,
		GasUsed:          sm.GasUsed,
		BaseFee:          sm.BaseFee,
		Status:           sm.Status,
		ErrorMessage:     sm.ErrorMessage,
	})
}

var ChainExCmd = &cli.Command{
	Name:  "chain",
	Usage: "Interact with filecoin blockchain",
	Subcommands: []*cli.Command{
		ChainGetBlockEX,
		ChainGetTipsetCmd,
	},
}

var ChainGetTipsetCmd = &cli.Command{
	Name:    "get-tipset",
	Aliases: []string{"gettipset"},
	Usage:   "View Tipset",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "height",
			Usage: "Get tipset according to altitude",
			Value: 1,
		},
	},
	Action: func(cctx *cli.Context) error {
		afmt := NewAppFmt(cctx.App)

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := ReqContext(cctx)

		height, err := strconv.ParseInt(cctx.Args().First(), 10, 64)
		if err != nil {
			return err
		}

		var ts *types.TipSet

		ts, err = api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(height), types.EmptyTSK)

		if err != nil {
			return err
		}

		if ts == nil {
			return errors.New("Tipset Not Found\n")
		}

		tipset := struct {
			Height    int64
			BlockCids []cid.Cid
		}{}

		tipset.Height = height
		if height == int64(ts.Height()) {
			tipset.BlockCids = ts.Cids()
		} else {
			tipset.BlockCids = nil
		}

		out, err := json.MarshalIndent(tipset, "", "  ")
		if err != nil {
			return err
		}

		afmt.Println(string(out))

		return nil
	},
}

var ChainGetBlockEX = &cli.Command{
	Name:      "get-block",
	Aliases:   []string{"getblock"},
	Usage:     "Get a block and print its details",
	ArgsUsage: "[blockhash]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "raw",
			Usage: "print just the raw block header",
		},
	},
	Action: func(cctx *cli.Context) error {
		afmt := NewAppFmt(cctx.App)

		api, closer, err := GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := ReqContext(cctx)

		if !cctx.Args().Present() {
			return fmt.Errorf("must pass cid of block to print")
		}

		blockhash := cctx.Args().First()
		bcid, err := cid.Decode(blockhash)
		if err != nil {
			return err
		}

		blk, err := api.ChainGetBlock(ctx, bcid)
		if err != nil {
			return xerrors.Errorf("get block failed: %w", err)
		}

		ts, err := api.ChainGetTipSetByHeight(ctx, blk.Height, types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("get block failed: %w", err)
		}
		if cctx.Bool("raw") {
			out, err := json.MarshalIndent(blk, "", "  ")
			if err != nil {
				return err
			}

			afmt.Println(string(out))
			return nil
		}

		msgs, err := api.ChainGetBlockMessages(ctx, bcid)
		if err != nil {
			return xerrors.Errorf("failed to get messages: %w", err)
		}
		pMsgs, err := api.ChainGetParentMessages(ctx, bcid)
		if err != nil {
			log.Warn(err)
			//return xerrors.Errorf("failed to get receipts: %w", err)
		}

		recpts, err := api.ChainGetParentReceipts(ctx, bcid)
		if err != nil {
			log.Warn(err)
			//return xerrors.Errorf("failed to get receipts: %w", err)
		}

		cblock := struct {
			types.BlockHeader
			BlsMessages    []*EXMessage
			SecpkMessages  []*EXSignedMessage
			ParentReceipts []*EXPerMessage
			BlockHash      string
		}{}

		blsMessages := make([]*EXMessage, 0)
		for _, msg := range msgs.BlsMessages {
			exmsg := new(EXMessage)
			exmsg.Message = *msg
			act, err := api.StateGetActor(ctx, msg.To, ts.Key())
			if err == nil {
				method, params, err := MethodAndParamsForMessage(msg.VMMessage(), act.Code)
				if err == nil {
					exmsg.MethodName = method
					if method == "CreateExternal" {
						exmsg.ParamJson = "{\"params\":" + "\"0x" + hex.EncodeToString(msg.VMMessage().Params)[6:len(hex.EncodeToString(msg.VMMessage().Params))-1] + "\"}"
					} else if method == "InvokeContract" {
						exmsg.ParamJson = "{\"params\":" + "\"0x" + hex.EncodeToString(msg.VMMessage().Params) + "\"}"
					} else {
						exmsg.ParamJson = params
					}
				} else {
					if msg.VMMessage().Method.String() == "3844450837" {
						exmsg.MethodName = "InvokeEVM"
					} else {
						exmsg.MethodName = "Unknow"
					}
				}

			}
			exmsg.BaseFee = blk.ParentBaseFee.Int64()
			blsMessages = append(blsMessages, exmsg)
		}

		secpkMessages := make([]*EXSignedMessage, 0)
		for _, msg := range msgs.SecpkMessages {
			exmsg := new(EXSignedMessage)
			exmsg.SignedMessage = *msg
			act, err := api.StateGetActor(ctx, msg.Message.To, ts.Key())
			if err == nil {
				method, params, err := MethodAndParamsForMessage(msg.VMMessage(), act.Code)
				if err == nil {
					exmsg.MethodName = method
					if method == "CreateExternal" {
						exmsg.ParamJson = "{\"params\":" + "\"0x" + hex.EncodeToString(msg.VMMessage().Params)[6:len(hex.EncodeToString(msg.VMMessage().Params))-1] + "\"}"
					} else if method == "InvokeContract" {
						exmsg.ParamJson = "{\"params\":" + "\"0x" + hex.EncodeToString(msg.VMMessage().Params) + "\"}"
					} else {
						exmsg.ParamJson = params
					}
				} else {
					if msg.VMMessage().Method.String() == "3844450837" {
						exmsg.MethodName = "InvokeEVM"
					} else {
						exmsg.MethodName = "Unknow"
					}
				}
			}
			exmsg.BaseFee = blk.ParentBaseFee.Int64()
			secpkMessages = append(secpkMessages, exmsg)
		}
		
		parentReceipts := make([]*EXPerMessage, 0)
		if len(pMsgs) == len(recpts){
			for i, recpt := range recpts {
				var exmsg = new(EXPerMessage);
				exmsg.CID = pMsgs[i].Cid
				act, err := api.StateGetActor(ctx, pMsgs[i].Message.To, types.EmptyTSK)
				if err == nil {
					method, _, err := MethodAndParamsForMessage(pMsgs[i].Message.VMMessage(), act.Code)
					if err == nil {
						exmsg.MethodName = method
						exmsg.GasUsed = recpt.GasUsed
						exmsg.Status = recpt.ExitCode.String()
						returnJson, _, err := ParseReturn(recpt.Return, pMsgs[i].Message.VMMessage().Method, act.Code)
						if err == nil {
							if method == "CreateExternal" {
								createExternalReturn := new(eam.CreateExternalReturn)
								if err := json.Unmarshal([]byte(returnJson), &createExternalReturn); err == nil {
									exCreateExternalReturn := ExCreateExternalReturn{
										ActorID:       createExternalReturn.ActorID,
										RobustAddress: createExternalReturn.RobustAddress,
										EthAddress:    "0x" + hex.EncodeToString(createExternalReturn.EthAddress[:]),
									}
									out, err := json.MarshalIndent(exCreateExternalReturn, "", "  ")
									if err != nil {
										exmsg.ReturnJson = returnJson
									} else {
										exmsg.ReturnJson = string(out)
									}
		
								} else {
									exmsg.ReturnJson = returnJson
								}
							} else if method == "InvokeContract" {
								exmsg.ReturnJson = "{\"return\":" + "\"0x" + hex.EncodeToString(recpt.Return) + "\"}"
							} else {
								exmsg.ReturnJson = returnJson
							}
						}
					}
					if eventsRoot := recpt.EventsRoot; eventsRoot != nil {
						api2, closer2, err := GetFullNodeAPIV1(cctx)
						if err != nil {
							return err
						}
						defer closer2()
	
						events, err := api2.ChainGetEvents(ctx, *eventsRoot)
						if err == nil {
							for _, evt := range events {
								var exEvent EXEvent
								exEvent.Address, _ = address.NewFromString("f0" + evt.Emitter.String())
								for _, e := range evt.Entries {
									exEvent.Topics = append(exEvent.Topics, EXEventEntry{e.Flags, e.Key, e.Codec, "0x" + hex.EncodeToString(e.Value)})
								}
								exmsg.Events = append(exmsg.Events, exEvent)
							}
						}
					}
					
				}
				parentReceipts = append(parentReceipts, exmsg);
			}
		}

		cblock.BlockHeader = *blk
		cblock.BlsMessages = blsMessages
		cblock.SecpkMessages = secpkMessages
		cblock.ParentReceipts = parentReceipts
		cblock.BlockHash = blockhash

		out, err := json.MarshalIndent(cblock, "", "  ")
		if err != nil {
			return err
		}

		afmt.Println(string(out))
		return nil
	},
}
