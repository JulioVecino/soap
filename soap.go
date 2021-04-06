package soap
import (
   	"encoding/xml"
   	"fmt"
   	"reflect"
    "bytes"
    "io/ioutil"
    "net/http"
    "net/url"
    "strings"
	"github.com/JulioVecino/logger"
)

// SoapClient return new *Client to handle the requests with the WSDL
func SoapClient(wsdl string, attrs map[string]string ) (*Client, error) {
	_, err := url.Parse(wsdl)
	if err != nil {
		return nil, err
	}
    soapPrefix := "soap"
    methodPrefix := "ws"
    for k, _ := range attrs {
        prefix := strings.Split(k, ":")
        if strings.Contains(prefix[1], "soa") {
            soapPrefix = prefix[1]
        } else {
            methodPrefix = prefix[1]
        }
    }

	c := &Client{
		wsdl:             wsdl,
		soapPrefix:       soapPrefix,
		methodPrefix:     methodPrefix,
		envelopeAttrs:    attrs,
	}

	return c, nil
}

type Client struct {
    request            []xml.Token
    soapPrefix         string
    methodPrefix       string
    envelopeAttrs      map[string]string
    wsdl               string
}

// Call call's the method with Params
func (c *Client) Call(method string,  params interface{}) ([]byte, error) {

    req, err := c.buildRequest(method, params)
    if err != nil {
        return nil, err
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    //Obtener el body del POST request hacia el servicio SOAP
    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

	return body, nil
}

func (c *Client) buildRequest(method string, params interface{}) (*http.Request, error) {
    // Inicio Envelop
    envelop := xml.StartElement{ Name: xml.Name{ Space: "",  Local: fmt.Sprintf("%s:Envelope", c.soapPrefix), }, }
    envelop.Attr = make([]xml.Attr, 0)
    for local, value := range c.envelopeAttrs {
        envelop.Attr = append(envelop.Attr, xml.Attr{ Name: xml.Name{Space: "", Local: local}, Value: value, })
    }

    // inicio Body
    body := xml.StartElement{ Name: xml.Name{ Space: "", Local: fmt.Sprintf("%s:Body", c.soapPrefix), }, }

    // inicio Ws method
    ws := xml.StartElement{ Name: xml.Name{ Space: "", Local: fmt.Sprintf("%s:%s", c.methodPrefix, method), }, }
    c.request = append(c.request, envelop, body, ws)

    // adding parameters
    t := reflect.TypeOf(params)
    if t.Kind() == reflect.Struct {
        v := reflect.ValueOf(params)
        numFields := v.NumField()
        for i := 0; i < numFields; i++ {
           name := t.Field(i).Tag.Get("xml")
           if (name != "")  {
               field := xml.StartElement{ Name: xml.Name{ Space: "", Local: name } }
               data := xml.CharData(fmt.Sprintf("%v", v.Field(i)))
               c.request = append(c.request, field, data, field.End())
           }
        }
    }
    // Fin ws method, Body, Envelop
    c.request = append(c.request, ws.End(), body.End(), envelop.End())

    doc := new(bytes.Buffer)
    enc := xml.NewEncoder(doc)
    enc.Indent("  ", "    ")
	for _, t := range c.request {
	    err := enc.EncodeToken(t)
		if err != nil {
			return nil, err
		}
	}
	err := enc.Flush();
	if err != nil {
		return nil, err
	}
    logger.Tittle("SOAP: XML REQUEST")
    logger.Txt(doc)
	r, err := http.NewRequest("POST", c.wsdl, doc)
	r.Header.Set("Content-type", "text/xml;charset=UTF-8")
	if err != nil {
		return nil, err
	}
	return r, nil
}

