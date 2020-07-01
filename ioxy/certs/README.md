# Certificate generation

## Generation script 

You can use the provided certificate generation script "genCerts.sh" to generate a custom root certificate and device 
certificates to install on the IoT device whose communication you want to intercept. 

Instructions for manual generation are also provided below.

Certificate generation will soon be implemented inside IOXY :)

## Cert tree with 3 devices

```
aws-certs
|-- ca
|   |-- rootCA.key
|   |-- rootCA.pem
|   `-- rootCA.srl
|-- devices
|   |-- d1
|   |   |-- d1.csr
|   |   |-- d1.key
|   |   `-- d1.pem
|   |-- d2
|   |   |-- d2.csr
|   |   |-- d2.key
|   |   `-- d2.pem
|   `-- d3
|       |-- d3.csr
|       |-- d3.key
|       `-- d3.pem
`-- verificationCert
    |-- verificationCert.csr
    |-- verificationCert.key
    `-- verificationCert.pem
```


## CA

    openssl genrsa -out rootCA.key 2048 && openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.pem

## VerificationCert

    openssl genrsa -out verificationCert.key 2048 && openssl req -new -key verificationCert.key -out verificationCert.csr && openssl x509 -req -in verificationCert.csr -CA ../ca/rootCA.pem -CAkey ../ca/rootCA.key -CAcreateserial -out verificationCert.pem -days 500 -sha256

## DeviceCert

    name=[DEV_NAME] && openssl genrsa -out $name.key 2048 && openssl req -new -key $name.key -out $name.csr && openssl x509 -req -in $name.csr -CA ../../ca/rootCA.pem -CAkey ../../ca/rootCA.key -CAcreateserial -out $name.pem -days 500 -sha256
