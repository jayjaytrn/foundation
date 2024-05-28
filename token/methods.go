package token

import (
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
)

// Metadata is a struct for metadata
type Metadata struct {
	Name            string          `json:"name"`
	Symbol          string          `json:"symbol"`
	Decimals        uint            `json:"decimals"`
	UnderlyingAsset string          `json:"underlying_asset"` //nolint:tagliatelle
	Issuer          string          `json:"issuer"`
	Methods         []string        `json:"methods"`
	TotalEmission   *big.Int        `json:"total_emission"` //nolint:tagliatelle
	Fee             *Fee            `json:"fee"`
	Rates           []*MetadataRate `json:"rates"`
}

// MetadataRate is a struct for rate
type MetadataRate struct {
	DealType string   `json:"deal_type"` //nolint:tagliatelle
	Currency string   `json:"currency"`
	Rate     *big.Int `json:"rate"`
	Min      *big.Int `json:"min"`
	Max      *big.Int `json:"max"`
}

// Fee is a struct for fee
type Fee struct {
	Address  string   `json:"address"`
	Currency string   `json:"currency"`
	Fee      *big.Int `json:"fee"`
	Floor    *big.Int `json:"floor"`
	Cap      *big.Int `json:"cap"`
}

// QueryMetadata returns Metadata
func (bt *BaseToken) QueryMetadata() (*Metadata, error) {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return &Metadata{}, err
	}
	m := &Metadata{
		Name:            bt.TokenConfig().GetName(),
		Symbol:          bt.ContractConfig().GetSymbol(),
		Decimals:        uint(bt.TokenConfig().GetDecimals()),
		UnderlyingAsset: bt.TokenConfig().GetUnderlyingAsset(),
		Issuer:          bt.TokenConfig().GetIssuer().GetAddress(),
		Methods:         bt.GetMethods(bt),
		TotalEmission:   new(big.Int).SetBytes(bt.config.GetTotalEmission()),
		Fee:             &Fee{},
	}

	if types.IsValidAddressLen(bt.config.GetFeeAddress()) {
		m.Fee.Address = types.AddrFromBytes(bt.config.GetFeeAddress()).String()
	}

	if bt.config.GetFee() != nil {
		m.Fee.Currency = bt.config.GetFee().GetCurrency()
		m.Fee.Fee = new(big.Int).SetBytes(bt.config.GetFee().GetFee())
		m.Fee.Floor = new(big.Int).SetBytes(bt.config.GetFee().GetFloor())
		m.Fee.Cap = new(big.Int).SetBytes(bt.config.GetFee().GetCap())
	}

	for _, r := range bt.config.GetRates() {
		m.Rates = append(m.Rates, &MetadataRate{
			DealType: r.GetDealType(),
			Currency: r.GetCurrency(),
			Rate:     new(big.Int).SetBytes(r.GetRate()),
			Min:      new(big.Int).SetBytes(r.GetMin()),
			Max:      new(big.Int).SetBytes(r.GetMax()),
		})
	}

	return m, nil
}

// QueryBalanceOf returns balance
func (bt *BaseToken) QueryBalanceOf(address *types.Address) (*big.Int, error) {
	return bt.TokenBalanceGet(address)
}

// QueryAllowedBalanceOf returns allowed balance
func (bt *BaseToken) QueryAllowedBalanceOf(address *types.Address, token string) (*big.Int, error) {
	return bt.AllowedBalanceGet(token, address)
}

// QueryLockedBalanceOf returns locked balance
func (bt *BaseToken) QueryLockedBalanceOf(address *types.Address) (*big.Int, error) {
	return bt.TokenBalanceGetLocked(address)
}

func (bt *BaseToken) QueryLockedAllowedBalanceOf(address *types.Address, token string) (*big.Int, error) {
	return bt.AllowedBalanceGetLocked(token, address)
}

// QueryDocumentsList - returns list of emission documents
func (bt *BaseToken) QueryDocumentsList() ([]core.Doc, error) {
	return core.DocumentsList(bt.GetStub())
}

// TxAddDocs - adds docs to a token
func (bt *BaseToken) TxAddDocs(sender *types.Sender, rawDocs string) error {
	if !sender.Equal(bt.Issuer()) {
		return errors.New("unathorized")
	}

	return core.AddDocs(bt.GetStub(), rawDocs)
}

// TxDeleteDoc - deletes doc from state
func (bt *BaseToken) TxDeleteDoc(sender *types.Sender, docID string) error {
	if !sender.Equal(bt.Issuer()) {
		return errors.New("unathorized")
	}

	return core.DeleteDoc(bt.GetStub(), docID)
}

// TxSetRate sets token rate to an asset for a type of deal
func (bt *BaseToken) TxSetRate(sender *types.Sender, dealType string, currency string, rate *big.Int) error {
	if !sender.Equal(bt.Issuer()) {
		return errors.New("unauthorized")
	}

	if rate.Sign() == 0 {
		return errors.New("trying to set rate = 0")
	}
	if bt.ContractConfig().GetSymbol() == currency {
		return errors.New("currency is equals token: it is impossible")
	}
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	for i, r := range bt.config.GetRates() {
		if r.GetDealType() == dealType && r.GetCurrency() == currency {
			bt.config.Rates[i].Rate = rate.Bytes()
			return bt.saveConfig()
		}
	}
	bt.config.Rates = append(bt.config.Rates, &proto.TokenRate{
		DealType: dealType,
		Currency: currency,
		Rate:     rate.Bytes(),
		Max:      new(big.Int).SetUint64(0).Bytes(),
		Min:      new(big.Int).SetUint64(0).Bytes(),
	})
	return bt.saveConfig()
}

// TxSetLimits sets limits for a deal type and an asset
func (bt *BaseToken) TxSetLimits(sender *types.Sender, dealType string, currency string, min *big.Int, max *big.Int) error {
	if !sender.Equal(bt.Issuer()) {
		return errors.New("unauthorized")
	}
	if min.Cmp(max) > 0 && max.Cmp(big.NewInt(0)) > 0 {
		return errors.New("min limit is greater than max limit")
	}
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	unknownDealType := true
	for i, r := range bt.config.GetRates() {
		if r.GetDealType() == dealType {
			unknownDealType = false
			if r.GetCurrency() == currency {
				bt.config.Rates[i].Max = max.Bytes()
				bt.config.Rates[i].Min = min.Bytes()
				return bt.saveConfig()
			}
		}
	}
	if unknownDealType {
		return fmt.Errorf("unknown DealType. Rate for deal type %s and currency %s was not set", dealType, currency)
	}
	return fmt.Errorf("unknown currency. Rate for deal type %s and currency %s was not set", dealType, currency)
}

// TxDeleteRate - deletes rate from state
func (bt *BaseToken) TxDeleteRate(sender *types.Sender, dealType string, currency string) error {
	if !sender.Equal(bt.Issuer()) {
		return errors.New("unauthorized")
	}
	if bt.ContractConfig().GetSymbol() == currency {
		return errors.New("currency is equals token: it is impossible")
	}
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	for i, r := range bt.config.GetRates() {
		if r.GetDealType() == dealType && r.GetCurrency() == currency {
			bt.config.Rates = append(bt.config.Rates[:i], bt.config.GetRates()[i+1:]...)
			return bt.saveConfig()
		}
	}

	return nil
}
