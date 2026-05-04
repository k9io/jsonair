/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

/* jsonair-encrypt encrypts or decrypts a configuration value for storage
   in the JSONAir database.

 Usage:

  # Encrypt
  echo -n "<base64-encoded-config>" | CONFIG_ENCRYPT_SECRET=<secret> ./jsonair-encrypt

  # Decrypt
  echo -n "<encrypted-value>" | CONFIG_ENCRYPT_SECRET=<secret> ./jsonair-encrypt -d

  The encrypted output is written to stdout and is ready to be inserted
  directly into the config_data column of the configurations table.

*/

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	cry "github.com/k9io/jsonair/internal/crypto"
	"github.com/joho/godotenv"
)

func main() {

	decrypt := flag.Bool("d", false, "decrypt instead of encrypt")
	flag.Parse()

	godotenv.Load()

	secret := os.Getenv("CONFIG_ENCRYPT_SECRET")

	if secret == "" {
		fmt.Fprintln(os.Stderr, "Error: CONFIG_ENCRYPT_SECRET environment variable is not set.")
		os.Exit(1)
	}

	key := cry.DeriveKey([]byte(secret))

	input, err := io.ReadAll(os.Stdin)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	if len(input) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input provided on stdin.")
		os.Exit(1)
	}

	if *decrypt {
		plaintext, err := cry.Decrypt(string(input), key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decrypting: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(plaintext))
	} else {
		encrypted, err := cry.Encrypt(input, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(encrypted)
	}

}
