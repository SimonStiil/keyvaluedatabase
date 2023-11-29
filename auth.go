package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	b64 "encoding/base64"
	"fmt"
	"log"
	"os"
)

type Auth struct {
	Users  map[string]User
	Config ConfigType
}
type User struct {
	PasswordEnc [32]byte
	Permissions ConfigPermissions
}

func AuthHash(data string) [32]byte {
	return sha256.Sum256([]byte(data))
}
func AuthTest(test [32]byte, fact [32]byte) bool {
	return (subtle.ConstantTimeCompare(test[:], fact[:]) == 1)
}
func AuthEncode(data [32]byte) string {
	return b64.StdEncoding.EncodeToString(data[:])
}
func AuthDecode(data string) [32]byte {
	bytes, err := b64.StdEncoding.DecodeString(data)

	if err != nil {
		panic(err)
	}
	if len(bytes) != 32 {
		panic(fmt.Sprintf("E wrong datalength %d should be 32", len(bytes)))
	}
	byteArray := [32]byte{}
	copy(byteArray[:], bytes)
	return byteArray
}
func (Auth *Auth) AuthGenerate(generate string, test string) {
	if generate != "" {
		hash := AuthHash(generate)
		encodedHash := AuthEncode(hash)
		if Auth.Config.Debug {
			log.Println("D encodedHash: ", encodedHash)
		} else {
			if test == "" {
				fmt.Println(encodedHash)
			}
		}
		if test != "" {
			testHash := AuthDecode(test)
			success := AuthTest(testHash, hash)
			fmt.Println("Test: ", success)
		}
		os.Exit(0)
	}
}
func (Auth *Auth) Init(config ConfigType) {
	Auth.Config = config
	Auth.Users = make(map[string]User)
	for i, v := range config.Users {
		if Auth.Config.Debug {
			log.Printf("D %v %v", i, v)
		}
		Auth.Users[v.Username] = AuthUnpack(v)
	}
	if Auth.Config.Debug {
		log.Printf("D auth.init - complete with %v users\n", len(Auth.Users))
	}
}
func AuthUnpack(Data ConfigUser) User {
	return User{
		PasswordEnc: AuthDecode(Data.Password),
		Permissions: Data.Permissions,
	}
}
func AuthPack(Name string, Data User) ConfigUser {
	return ConfigUser{
		Username:    Name,
		Password:    AuthEncode(Data.PasswordEnc),
		Permissions: Data.Permissions,
	}
}
func AuthTestPermission(permission ConfigPermissions, expected ConfigPermissions) bool {
	return ((expected.Read == false || permission.Read) &&
		(expected.Write == false || permission.Write) &&
		(expected.List == false || permission.List))
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func AuthGenerateRandomString(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	StringAsBytes := make([]byte, length)
	for i, b := range bytes {
		location := int(b) % len(charset)
		char := charset[location]
		StringAsBytes[i] = char
	}
	return string(StringAsBytes)
}
