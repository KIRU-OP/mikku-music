from mikku.core.bot import Anony
from mikku.core.dir import dirr
from mikku.core.git import git
from mikku.core.userbot import Userbot
from mikku.misc import heroku, dbb

from .logging import LOGGER

dirr()
git()
dbb()
heroku()

app = Anony()
userbot = Userbot()

from .platforms import *

Apple = AppleAPI()
Carbon = CarbonAPI()
SoundCloud = SoundAPI()
Spotify = SpotifyAPI()
Resso = RessoAPI()
Telegram = TeleAPI()
YouTube = YouTubeAPI()
