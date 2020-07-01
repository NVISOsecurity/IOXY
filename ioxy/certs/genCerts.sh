#!/bin/sh

DEVICE_NAME="d1"

# create directories if needed
mkdir -p "./ca"
mkdir -p "./devices/$DEVICE_NAME"
mkdir -p "./verificationCert"

# generate root key 
echo "=============------------============"
echo "[+] Generating Root cert"
echo "=============------------============"
echo ""
cd "./ca"
openssl genrsa -out rootCA.key 2048 && openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.pem
cd .. 

# generate verification cert
echo ""
echo "=============------------============"
echo "[+] Generating Verification cert"
echo "=============------------============"
echo ""
cd "./verificationCert"
openssl genrsa -out verificationCert.key 2048 && openssl req -new -key verificationCert.key -out verificationCert.csr && openssl x509 -req -in verificationCert.csr -CA ../ca/rootCA.pem -CAkey ../ca/rootCA.key -CAcreateserial -out verificationCert.pem -days 500 -sha256
cd ..

# generate device cert
echo ""
echo "=============------------============"
echo "[+] Generating Device cert"
echo "=============------------============"
echo ""
cd "./devices/$DEVICE_NAME"
openssl genrsa -out $DEVICE_NAME.key 2048 && openssl req -new -key $DEVICE_NAME.key -out $DEVICE_NAME.csr && openssl x509 -req -in $DEVICE_NAME.csr -CA ../../ca/rootCA.pem -CAkey ../../ca/rootCA.key -CAcreateserial -out $DEVICE_NAME.pem -days 500 -sha256

echo ""
echo "=============------------============"
echo "[+] Done!"
echo "=============------------============"
echo ""