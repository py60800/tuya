// Copyright 2019 py60800.
// Use of this source code is governed by Apache-2 licence
// license that can be found in the LICENSE file.

package tuya

import (
   "crypto/aes"
   "crypto/md5"
   "encoding/base64"
   "encoding/hex"
   "errors"
   //"log"
)

func md5Sign(b []byte, key []byte, version string) []byte {
   h := md5.New()
   h.Write([]byte("data="))
   h.Write(b)
   h.Write([]byte("||lpv=" + version + "||"))
   h.Write(key)
   hash := h.Sum(nil)
   return []byte(hex.EncodeToString(hash[4:12]))
}
func aesEncrypt(data []byte, key []byte) ([]byte, error) {
   block, err := aes.NewCipher([]byte(key))
   if err != nil {
      return nil, err
   }
   bs := block.BlockSize()
   remain := len(data) % bs
   if remain == 0 {
      remain = bs
   }
   padd := make([]byte, bs-remain)
   for i := range padd {
      padd[i] = byte(bs - remain)
   }
   data = append(data, padd...)
   ciphertext := make([]byte, len(data))
   for i := 0; i < len(data); i = i + bs {
      block.Encrypt(ciphertext[i:i+bs], data[i:i+bs])
   }
   return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

func aesDecrypt(data []byte, key []byte) ([]byte, error) {
   n := base64.StdEncoding.DecodedLen(len(data))
   ciphertext := make([]byte, n)
   nc, er1 := base64.StdEncoding.Decode(ciphertext, data)
   if er1 != nil {
      return []byte{}, er1
   }
   ciphertext = ciphertext[:nc]
   block, er2 := aes.NewCipher([]byte(key))
   if er2 != nil {
      return []byte{}, er2
   }
   bs := block.BlockSize()
   if nc%bs != 0 && nc < 16 {
      return []byte{}, errors.New("Bad ciphertext len")
   }
   cleartext := make([]byte, nc)
   for i := 0; i < nc; i = i + bs {
      block.Decrypt(cleartext[i:i+bs], ciphertext[i:i+bs])
   }
   // remove padding
   p := int(cleartext[nc-1])
   if p < 0 || p > bs {
      return []byte{}, errors.New("Bad padding")
   }
   return cleartext[:nc-p], nil
}
