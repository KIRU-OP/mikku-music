<div align="center">

<img src="https://litter.catbox.moe/vofelg.jpg" width="1000" alt="Mikku Music"/>

# 🎧 MIKKU MUSIC

**Fast • Powerful • Premium Telegram Music Bot**

[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?style=for-the-badge&logo=python&logoColor=white)](https://www.python.org/)
[![Pyrogram](https://img.shields.io/badge/Pyrogram-v2-0088CC?style=for-the-badge&logo=telegram&logoColor=white)](https://github.com/KurimuzonAkuma/pyrogram)
[![MongoDB](https://img.shields.io/badge/MongoDB-Atlas-47A248?style=for-the-badge&logo=mongodb&logoColor=white)](https://cloud.mongodb.com)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)

</div>

---

## ✨ Features

🎵 YouTube • Spotify • Apple Music • SoundCloud • Resso • Telegram files  
🎛️ Pause • Resume • Skip • Seek • Loop • Shuffle • Speed  
👥 Multi-assistant support (up to 5) &nbsp;|&nbsp; 🛡️ Sudo, Gban, Block, Maintenance mode  

---

## 🔑 Required Variables

| Variable | Source |
|---|---|
| `API_ID` / `API_HASH` | [my.telegram.org](https://my.telegram.org) |
| `BOT_TOKEN` | [@BotFather](https://t.me/BotFather) |
| `MONGO_DB_URI` | [MongoDB Atlas](https://cloud.mongodb.com) |
| `STRING_SESSION` | Pyrogram **v2** session |
| `OWNER_ID` | [@userinfobot](https://t.me/userinfobot) |
| `LOGGER_ID` | Private log group ID (`-100xxxxxxxxxx`) |

---

## 🚀 Deploy

### 🖥️ VPS (Bash Setup)

```bash
sudo apt update && sudo apt install git tmux -y
git clone https://github.com/KIRU-OP/mikku-music.git
cd mikku-music
bash setup        # installs everything + creates .env
tmux new -s mikku
bash start        # run the bot (Detach: Ctrl+B, D)
```

### 🐳 Docker

```bash
git clone https://github.com/KIRU-OP/mikku-music.git && cd mikku-music
nano .env         # fill the required variables
docker build -t mikku-music .
docker run -d --name mikku --restart unless-stopped --env-file .env mikku-music
```

### ☁️ Heroku

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/KIRU-OP/mikku-music)

---

## 🔄 Update

```bash
git pull && pip3 install -U -r requirements.txt
```

---

## 💬 Support

| | |
|---|---|
| 📢 **Channel** | [˹ɴᴇxᴏ ʙᴏᴛ sᴜᴘᴘᴏʀᴛ˼](https://t.me/NEXO_BOT_SUPPORT) |
| 💗 **Group** | [୨୧ ɴᴇxᴏ ᴏғғɪᴄɪᴀʟ ᴄʜᴀᴛ 💗 💌](https://t.me/group_voice_chat) |
| 👑 **Owner** | [𝐕 𝚬 𝚴 𝚶 𝚳 ⴕ](https://t.me/about_me_venom) |

---

<div align="center">

⭐ **Pasand aaya to Star zaroor dena!** ⭐

Made with ❤️ by [KIRU-OP](https://github.com/KIRU-OP)

</div>
