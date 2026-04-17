<div align="center">
  <h1 align="center">Reg-X</h1>
  <p><strong>A Highly Secure, Fast, and Beautiful Personal File Encrypter</strong></p>
</div>

<p align="center">
  Reg-X (formerly Personal Secure Encrypter) is a blazing-fast, offline-first desktop application designed to secure your sensitive files and folders using industry-standard cryptography. Built with a modern, glassmorphism UI overlaying an animated gradient background.
</p>

---

## 🌟 Features

- **Uncompromising Security:** Powered by **AES-256-GCM** authenticated encryption and **Argon2id** key derivation.
- **Bulk Processing:** Encrypt or decrypt entire folders or select multiple files at once.
- **High Performance:** Go-powered backend gives incredibly fast encryption and decryption speeds, even for huge files (up to 2GB memory-safety limit).
- **Stunning UI:** Built with React, Tailwind CSS, providing an elegant Dark Mode with animated gradient vibes.
- **Fully Offline & Local:** Absolutely no data, telemetry, or passwords are ever sent to any remote server. Your keys only exist in RAM during the operation and are explicitly wiped afterwards.

---

## 🔒 Security Specifications

- **Key Derivation:** Argon2id (OWASP recommended parameters: 64MB memory, 3 iterations, 4 threads).
- **Encryption Algorithm:** AES-256-GCM (Provides both confidentiality and integrity).
- **Unique Salts & Nonces:** Every single file receives a cryptographically secure 16-byte random salt and 12-byte random nonce.
- **Tamper-Proof:** Any modification to the encrypted file will immediately fail the GCM authentication tag check.
- **Memory Hardening:** Passwords and plaintext are actively zeroed out from memory immediately after execution to prevent memory scraping.

---

## 🛠️ Prerequisites

To build and run this project from source, you will need:

1. [Go (1.20+)](https://go.dev/dl/)
2. [Node.js (18+)](https://nodejs.org/) & NPM
3. [Wails CLI (v2)](https://wails.io/docs/gettingstarted/installation)

Install the Wails CLI:
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

---

## 🚀 Installation & Building

1. **Clone the repository:**
   ```bash
   git clone https://github.com/JustHackedOn/Reg-X.git
   cd Reg-X
   ```

2. **Run in Development Mode:**
   This starts the app with live-reloading enabled for both the React frontend and Go backend.
   ```bash
   wails dev
   ```

3. **Build the Final Production Binary:**
   This produces a single, standalone executable that requires no installation.
   ```bash
   wails build -clean
   ```
   **Output:** The compiled executable will be located in `build/bin/Reg-X.exe`.

---

## 📖 How to Use

1. **Launch Reg-X.**
2. Go to the **Encrypt** or **Decrypt** tab on the left sidebar.
3. Click **"Choose Files"** or **"Choose Folder"** or drag and drop your files into the main window.
4. Set a strong password (minimum 8 characters).
5. Click **"Start Encrypt"** / **"Start Decrypt"**. 
6. Your files are instantly processed! By default, the output files are heavily encrypted files with the `.pse` extension saved directly next to the original files.

> **Tip:** You can configure a custom default output folder and enable "Delete originals after encryption" in the **Settings** menu!

---

## 👨‍💻 Developed by

Crafted with ❤️ by **[JustHackedOn](https://github.com/JustHackedOn)**. 

### Disclaimer
*This software is provided "as is", without warranty of any kind. Please make sure you do not forget your encryption passwords. There is absolutely no way to recover your data if the password is lost.*
