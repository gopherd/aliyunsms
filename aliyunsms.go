package aliyunsms

import (
	"encoding/json"
	"errors"
	"net/url"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/gopherd/doge/query"
	"github.com/gopherd/doge/sms"
)

func init() {
	sms.Register("aliyun", open)
}

func open(source string) (sms.Provider, error) {
	var (
		options Options
		err     = parseSource(&options, source)
	)
	if err != nil {
		return nil, err
	}
	return NewClient(options)
}

type Options struct {
	Scheme string `json:"scheme"`
	Domain string `json:"domain"`

	Version      string `json:"version"`
	ApiName      string `json:"api_name"`
	AccessKey    string `json:"access_key"`
	AccessSecret string `json:"access_secret"`
	SignName     string `json:"sign_name"`
	TemplateCode string `json:"template_code"`

	RegionId string `json:"region_id"`
}

func (options Options) String() string {
	u := url.URL{
		Scheme: options.Scheme,
		Host:   options.Domain,
		RawQuery: url.Values{
			"version":       {options.Version},
			"api_name":      {options.ApiName},
			"access_key":    {options.AccessKey},
			"access_secret": {options.AccessSecret},
			"sign_name":     {options.SignName},
			"template_code": {options.TemplateCode},
			"region_id":     {options.RegionId},
		}.Encode(),
	}
	return u.String()
}

// parseSource parses options source string. Formats of source:
//
//	$scheme://$domain?k1=v1&k2=v2&...&kn=vn
//
func parseSource(options *Options, source string) error {
	u, err := url.Parse(source)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		return errors.New("scheme required")
	}
	options.Scheme = u.Scheme
	options.Domain = u.Host
	return query.New(query.Query(u.Query())).
		RequiredString(&options.Version, "version").
		RequiredString(&options.ApiName, "api_name").
		RequiredString(&options.AccessKey, "access_key").
		RequiredString(&options.AccessSecret, "access_secret").
		RequiredString(&options.SignName, "sign_name").
		RequiredString(&options.TemplateCode, "template_code").
		String(&options.RegionId, "region_id", "default").
		Err()
}

type Client struct {
	options   Options
	sdkClient *sdk.Client
}

func NewClient(options Options) (*Client, error) {
	sdkClient, err := sdk.NewClientWithAccessKey(options.RegionId, options.AccessKey, options.AccessSecret)
	if err != nil {
		return nil, err
	}
	return &Client{
		options:   options,
		sdkClient: sdkClient,
	}, nil
}

// SendCode implements sms.Provider SendCode method
func (c *Client) SendCode(phoneNumber, code string) error {
	req := requests.NewCommonRequest()
	req.Method = "POST"
	req.Scheme = c.options.Scheme
	req.Domain = c.options.Domain
	req.Version = c.options.Version
	req.ApiName = c.options.ApiName
	req.QueryParams["SignName"] = c.options.SignName
	req.QueryParams["TemplateCode"] = c.options.TemplateCode
	req.QueryParams["PhoneNumbers"] = phoneNumber

	param, err := json.Marshal(struct {
		Code string `json:"code"`
	}{Code: code})
	if err != nil {
		return err
	}
	req.QueryParams["TemplateParam"] = string(param)

	res, err := c.sdkClient.ProcessCommonRequest(req)
	if err != nil {
		return err
	}
	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(res.GetHttpContentBytes(), &result); err != nil {
		return err
	}
	if result.Code != "OK" {
		return errors.New(result.Message)
	}
	return nil
}
