import os
import re
import math
import aiofiles
import aiohttp
import logging
from PIL import Image, ImageDraw, ImageEnhance, ImageFilter, ImageFont, ImageOps
from youtubesearchpython.__future__ import VideosSearch
from config import YOUTUBE_IMG_URL
from mikku import app

logging.basicConfig(level=logging.ERROR, format='%(asctime)s - %(levelname)s - %(message)s')

CACHE_DIR = "cache"
os.makedirs(CACHE_DIR, exist_ok=True)

FONT_BOLD = "mikku/assets/thumb/font2.ttf"
FONT_REG = "mikku/assets/thumb/font.ttf"

# ---------- Layout ----------
W, H = 1280, 720

CARD_W, CARD_H = 1000, 360
CARD_X = (W - CARD_W) // 2
CARD_Y = 150
CARD_R = 70

ART = 260
ART_X = CARD_X + 45
ART_Y = CARD_Y + (CARD_H - ART) // 2
ART_R = 30

TEXT_X = ART_X + ART + 50
TEXT_MAX_W = CARD_X + CARD_W - 60 - TEXT_X

LABEL_COLOR = (200, 215, 235, 255)      # "NOW PLAYING" / artist (light blue-grey)
WHITE = (255, 255, 255, 255)


def load_font(path, size):
    try:
        return ImageFont.truetype(path, size)
    except Exception:
        return ImageFont.load_default()


def trim_to_width(text: str, font, max_w: int) -> str:
    ellipsis = "…"
    text = text[:50]
    if font.getlength(text) <= max_w:
        return text
    for i in range(len(text) - 1, 0, -1):
        if font.getlength(text[:i] + ellipsis) <= max_w:
            return text[:i] + ellipsis
    return ellipsis


def spaced_text(draw, xy, text, font, fill, spacing=3):
    x, y = xy
    for ch in text:
        draw.text((x, y), ch, font=font, fill=fill)
        x += font.getlength(ch) + spacing
    return x


def draw_star(draw, cx, cy, r, fill):
    pts = []
    for i in range(10):
        ang = -math.pi / 2 + i * math.pi / 5
        rad = r if i % 2 == 0 else r * 0.45
        pts.append((cx + rad * math.cos(ang), cy + rad * math.sin(ang)))
    draw.polygon(pts, fill=fill)


def draw_music_note(draw, x, y, fill=WHITE):
    draw.ellipse((x, y + 34, x + 18, y + 50), fill=fill)
    draw.ellipse((x + 28, y + 28, x + 46, y + 44), fill=fill)
    draw.rectangle((x + 14, y + 8, x + 18, y + 42), fill=fill)
    draw.rectangle((x + 42, y + 2, x + 46, y + 36), fill=fill)
    draw.polygon([(x + 14, y + 8), (x + 46, y + 2), (x + 46, y + 14), (x + 14, y + 20)], fill=fill)


def rounded_mask(size, radius):
    m = Image.new("L", size, 0)
    ImageDraw.Draw(m).rounded_rectangle((0, 0, size[0] - 1, size[1] - 1), radius, fill=255)
    return m


def vertical_gradient(size, top, bottom):
    w, h = size
    top_img = Image.new("RGBA", size, top)
    bot_img = Image.new("RGBA", size, bottom)
    mask = Image.linear_gradient("L").resize((w, h))
    return Image.composite(bot_img, top_img, mask)


async def get_thumb(videoid: str) -> str:
    cache_path = os.path.join(CACHE_DIR, f"{videoid}_v6.png")
    if os.path.exists(cache_path):
        return cache_path

    try:
        results = VideosSearch(f"https://www.youtube.com/watch?v={videoid}", limit=1)
        results_data = await results.next()
        data = results_data["result"][0]
        title = re.sub(r"\W+", " ", data.get("title", "Unsupported Title")).title()
        thumbnail = data.get("thumbnails", [{}])[0].get("url", YOUTUBE_IMG_URL)
        duration = data.get("duration")
        views = data.get("viewCount", {}).get("short", "Unknown Views")
        channel = (data.get("channel") or {}).get("name", "Unknown Artist")
    except Exception as e:
        logging.error(f"Error fetching YouTube data: {e}")
        title, thumbnail, duration, views = "Unsupported Title", YOUTUBE_IMG_URL, None, "Unknown Views"
        channel = "Unknown Artist"

    is_live = not duration or str(duration).strip().lower() in {"", "live", "live now"}
    duration_text = "LIVE" if is_live else duration

    thumb_path = os.path.join(CACHE_DIR, f"thumb_{videoid}.jpg")
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(thumbnail) as resp:
                if resp.status == 200:
                    async with aiofiles.open(thumb_path, "wb") as f:
                        await f.write(await resp.read())
                else:
                    logging.error(f"Failed to download thumbnail (HTTP {resp.status})")
                    return YOUTUBE_IMG_URL
    except Exception as e:
        logging.error(f"Download error: {e}")
        return YOUTUBE_IMG_URL

    try:
        base = Image.open(thumb_path).convert("RGBA")
        bg_src = base.resize((W, H))
        bg = ImageEnhance.Brightness(bg_src.filter(ImageFilter.GaussianBlur(30))).enhance(0.55)
        bg = ImageEnhance.Color(bg).enhance(1.2)
    except Exception as e:
        logging.error(f"Image processing error: {e}")
        return YOUTUBE_IMG_URL

    # ---------- Card shadow ----------
    try:
        shadow = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        ImageDraw.Draw(shadow).rounded_rectangle(
            (CARD_X, CARD_Y + 18, CARD_X + CARD_W, CARD_Y + CARD_H + 18),
            CARD_R, fill=(0, 0, 0, 140))
        shadow = shadow.filter(ImageFilter.GaussianBlur(28))
        bg = Image.alpha_composite(bg, shadow)
    except Exception as e:
        logging.error(f"Shadow error: {e}")

    # ---------- Glass card ----------
    try:
        area = bg.crop((CARD_X, CARD_Y, CARD_X + CARD_W, CARD_Y + CARD_H))
        area = area.filter(ImageFilter.GaussianBlur(10))
        tint = vertical_gradient((CARD_W, CARD_H), (95, 120, 150, 150), (30, 45, 70, 215))
        glass = Image.alpha_composite(area, tint)

        # top shine
        shine = Image.new("RGBA", (CARD_W, CARD_H), (0, 0, 0, 0))
        ImageDraw.Draw(shine).ellipse((-200, -CARD_H, CARD_W + 200, CARD_H // 2),
                                      fill=(255, 255, 255, 22))
        glass = Image.alpha_composite(glass, shine)

        # border
        border = Image.new("RGBA", (CARD_W, CARD_H), (0, 0, 0, 0))
        ImageDraw.Draw(border).rounded_rectangle((0, 0, CARD_W - 1, CARD_H - 1), CARD_R,
                                                 outline=(255, 255, 255, 90), width=2)
        glass = Image.alpha_composite(glass, border)

        bg.paste(glass, (CARD_X, CARD_Y), rounded_mask((CARD_W, CARD_H), CARD_R))
    except Exception as e:
        logging.error(f"Card error: {e}")

    # ---------- Album art ----------
    try:
        art_shadow = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        ImageDraw.Draw(art_shadow).rounded_rectangle(
            (ART_X, ART_Y + 10, ART_X + ART, ART_Y + ART + 10), ART_R, fill=(0, 0, 0, 150))
        art_shadow = art_shadow.filter(ImageFilter.GaussianBlur(14))
        bg = Image.alpha_composite(bg, art_shadow)

        art = ImageOps.fit(base, (ART, ART), method=Image.Resampling.LANCZOS)
        bg.paste(art, (ART_X, ART_Y), rounded_mask((ART, ART), ART_R))
    except Exception as e:
        logging.error(f"Art error: {e}")

    # ---------- UI layer (text, pill, icons) ----------
    try:
        ui = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        d = ImageDraw.Draw(ui)

        small_font = load_font(FONT_REG, 26)
        title_font = load_font(FONT_BOLD, 50)
        artist_font = load_font(FONT_REG, 38)
        pill_font = load_font(FONT_BOLD, 32)
        meta_font = load_font(FONT_REG, 24)
        label_font = load_font(FONT_REG, 28)

        # NOW PLAYING
        spaced_text(d, (TEXT_X, CARD_Y + 52), "NOW PLAYING", small_font, LABEL_COLOR, spacing=3)

        # Title + star
        title_text = trim_to_width(title, title_font, TEXT_MAX_W - 50)
        ty = CARD_Y + 92
        d.text((TEXT_X, ty), title_text, font=title_font, fill=WHITE)
        tw = title_font.getlength(title_text)
        draw_star(d, TEXT_X + tw + 28, ty + 33, 13, (190, 200, 215, 255))

        # Artist / channel
        artist_text = trim_to_width(channel, artist_font, TEXT_MAX_W)
        d.text((TEXT_X, CARD_Y + 164), artist_text, font=artist_font, fill=LABEL_COLOR)

        # Play pill
        pill_w, pill_h = 210, 72
        px, py = TEXT_X, CARD_Y + 238
        d.rounded_rectangle((px, py, px + pill_w, py + pill_h), pill_h // 2,
                            fill=(255, 255, 255, 55))
        cy = py + pill_h // 2
        d.polygon([(px + 32, cy - 15), (px + 32, cy + 15), (px + 58, cy)], fill=WHITE)
        d.text((px + 78, cy - 20), "Play", font=pill_font, fill=WHITE)

        # views | duration next to pill
        mx = px + pill_w + 28
        if is_live:
            d.ellipse((mx, cy - 8, mx + 16, cy + 8), fill=(255, 59, 48, 255))
            d.text((mx + 26, cy - 15), "LIVE", font=meta_font, fill=(255, 90, 80, 255))
            d.text((mx + 100, cy - 15), f"|  {views}", font=meta_font, fill=LABEL_COLOR)
        else:
            d.text((mx, cy - 15), f"{duration_text}  |  {views}", font=meta_font, fill=LABEL_COLOR)

        # Music note (top-right)
        draw_music_note(d, CARD_X + CARD_W - 105, CARD_Y + 38)

        # Bottom label
        label = f"Music  •  @{app.username}"
        lw = label_font.getlength(label)
        d.text(((W - lw) / 2, CARD_Y + CARD_H + 28), label, font=label_font,
               fill=(235, 240, 250, 255))

        bg = Image.alpha_composite(bg, ui)
    except Exception as e:
        logging.error(f"Text rendering error: {e}")

    # ---------- Save & cleanup ----------
    try:
        os.remove(thumb_path)
    except Exception as e:
        logging.error(f"Cleanup error: {e}")

    try:
        bg.convert("RGB").save(cache_path, quality=95)
        return cache_path
    except Exception as e:
        logging.error(f"Save error: {e}")
        return YOUTUBE_IMG_URL
