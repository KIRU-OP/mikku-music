package app

import (
 "context"
 "encoding/json"
 "fmt"
 "log"
 "math"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"
 "sync"

 "github.com/go-telegram/bot"
 "github.com/go-telegram/bot/models"
 "github.com/KIRU-OP/mikku-music/internal/music"
)

type featureData struct {
 Languages map[int64]string `json:"languages"`
 PlayModes map[int64]string `json:"play_modes"`
 Loops map[int64]int `json:"loops"`
 Speeds map[int64]float64 `json:"speeds"`
 Sudoers map[int64]bool `json:"sudoers"`
 BannedUsers map[int64]bool `json:"banned_users"`
 Chats map[int64]bool `json:"chats"`
 Maintenance bool `json:"maintenance"`
}
var featureMu sync.Mutex
var features = newFeatureData()
func newFeatureData() featureData { return featureData{Languages:map[int64]string{},PlayModes:map[int64]string{},Loops:map[int64]int{},Speeds:map[int64]float64{},Sudoers:map[int64]bool{},BannedUsers:map[int64]bool{},Chats:map[int64]bool{}} }
func featurePath() string { return envOr("FEATURE_STORE_PATH","data/features.json") }
func (s *Service) loadFeatureData() {
 featureMu.Lock(); defer featureMu.Unlock()
 raw,err:=os.ReadFile(featurePath()); if err!=nil { if !os.IsNotExist(err){log.Printf("read feature store: %v",err)}; return }
 d:=newFeatureData(); if err=json.Unmarshal(raw,&d); err!=nil {log.Printf("decode feature store: %v",err);return}
 if d.Languages==nil {d.Languages=map[int64]string{}}; if d.PlayModes==nil {d.PlayModes=map[int64]string{}}; if d.Loops==nil {d.Loops=map[int64]int{}}; if d.Speeds==nil {d.Speeds=map[int64]float64{}}; if d.Sudoers==nil {d.Sudoers=map[int64]bool{}}; if d.BannedUsers==nil {d.BannedUsers=map[int64]bool{}}; if d.Chats==nil {d.Chats=map[int64]bool{}}
 features=d
}
func saveFeatureDataLocked() error {
 path:=featurePath(); if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return err}
 raw,err:=json.MarshalIndent(features,"","  ");if err!=nil{return err}
 tmp:=path+".tmp"; if err=os.WriteFile(tmp,raw,0600);err!=nil{return err};return os.Rename(tmp,path)
}
func (s *Service) isSudo(id int64) bool { featureMu.Lock(); defer featureMu.Unlock(); return features.Sudoers[id] }
func (s *Service) isFeatureBanned(user *models.User) bool { if user==nil || isOwner(user.ID) || s.isSudo(user.ID) {return false}; featureMu.Lock(); defer featureMu.Unlock(); return features.BannedUsers[user.ID] }
func (s *Service) rememberChat(chat int64) {
 featureMu.Lock(); defer featureMu.Unlock()
 if features.Chats[chat] { return }
 features.Chats[chat]=true
 if err:=saveFeatureDataLocked();err!=nil {log.Printf("save known chats: %v",err)}
}
func (s *Service) isMaintenanceEnabled() bool { featureMu.Lock(); defer featureMu.Unlock(); return features.Maintenance }
func (s *Service) isAdminPlayMode(chat int64) bool { featureMu.Lock(); defer featureMu.Unlock(); return strings.EqualFold(features.PlayModes[chat],"admins") }
func (s *Service) canPlayInChat(chat,user int64) bool {
 if user==0 { return !s.isAdminPlayMode(chat) }
 if isOwner(user)||s.isSudo(user)||s.canControlChat(chat,user) { return true }
 return !s.isAdminPlayMode(chat)
}
func (s *Service) handleExtendedCommand(ctx context.Context,b *bot.Bot,msg *models.Message,command,args string,respond func(string)) bool {
 if msg==nil{return false}
 id:=int64(0); if msg.From!=nil{id=msg.From.ID}; chat:=msg.Chat.ID
 switch command {
 case "/lang","/setlang","/language":
  if args=="" { featureMu.Lock(); v:=features.Languages[chat]; featureMu.Unlock(); if v==""{v="en"}; respond("🌐 Language: "+v+"\nAvailable: en, hi, pa, ba\nUse /lang <en|hi|pa|ba>."); return true }
  v:=strings.ToLower(strings.TrimSpace(args)); if v!="en"&&v!="hi"&&v!="pa"&&v!="ba"{respond("Unsupported language. Available: en, hi, pa, ba.");return true}
  featureMu.Lock(); features.Languages[chat]=v; err:=saveFeatureDataLocked(); featureMu.Unlock(); if err!=nil{respond("Could not save language: "+short(err.Error(),120))}else{respond("✅ Language set to "+v+".")}; return true
 case "/playmode","/mode":
  featureMu.Lock(); defer featureMu.Unlock()
  if args=="" {v:=features.PlayModes[chat];if v==""{v="Everyone"};respond("🎛️ Play mode: "+v+"\nUse /mode everyone or /mode admins.");return true}
  v:=strings.ToLower(strings.TrimSpace(args)); if v!="everyone"&&v!="admins"{respond("Usage: /mode everyone|admins");return true}
  if id==0 || (!isOwner(id)&&!features.Sudoers[id]&&!s.adminChatLocked(chat,id)){respond("⛔ Only the owner/authorized chat admin can change play mode.");return true}
  if v=="everyone"{features.PlayModes[chat]="Everyone"}else{features.PlayModes[chat]="Admins"}
  if err:=saveFeatureDataLocked();err!=nil{respond("Could not save setting: "+short(err.Error(),120))}else{respond("✅ Play mode set to "+v+".")};return true
 case "/loop","/cloop":
  if args==""{respond("Usage: /loop <0-10> (0 disables repeating; 1-10 repeats the current track that many extra times).");return true}
  n,err:=strconv.Atoi(strings.TrimSpace(args));if err!=nil||n<0||n>10{respond("Usage: /loop <0-10>");return true}
  featureMu.Lock();features.Loops[chat]=n;err=saveFeatureDataLocked();featureMu.Unlock()
  if err!=nil{respond("Could not save loop setting: "+short(err.Error(),120))}else if n==0{respond("🔁 Repeat disabled.")}else{respond(fmt.Sprintf("🔁 Current track will repeat %d more time(s).",n))};return true
 case "/shuffle","/cshuffle":
  if id==0||(!isOwner(id)&&!s.canControlChat(chat,id)&&!s.isSudo(id)){respond("⛔ You are not authorized to shuffle this queue.");return true}
  n:=s.queues.ShuffleUpcoming(chat);if n==0{respond("Queue is empty.")}else if n<3{respond("Not enough upcoming tracks to shuffle; the current track remains first.")}else{respond(fmt.Sprintf("🔀 Shuffled %d queued track(s); current track remains first.",n))};return true
 case "/speed","/cspeed","/slow","/cslow","/playback","/cplayback":
  if args=="" {featureMu.Lock();v:=features.Speeds[chat];featureMu.Unlock();if v==0{v=1};respond(fmt.Sprintf("⏩ Playback speed: %.2gx\nUse /speed 0.5, 0.75, 1, 1.25, 1.5, or 2. Speed is applied to downloaded audio before it is sent.",v));return true}
  v,err:=strconv.ParseFloat(strings.TrimSpace(args),64);if err!=nil||v<0.5||v>2||!allowedSpeed(v){respond("Usage: /speed 0.5|0.75|1|1.25|1.5|2");return true}
  if id==0||(!isOwner(id)&&!s.canControlChat(chat,id)&&!s.isSudo(id)){respond("⛔ You are not authorized to change playback settings.");return true}
  featureMu.Lock();features.Speeds[chat]=v;err=saveFeatureDataLocked();featureMu.Unlock();if err!=nil{respond("Could not save speed: "+short(err.Error(),120))}else{respond(fmt.Sprintf("✅ Speed set to %.2gx for upcoming downloads.",v))};return true
 case "/broadcast":
  if id==0||(!isOwner(id)&&!s.isSudo(id)){respond("⛔ Owner/sudo access required.");return true}
  if strings.TrimSpace(args)==""{respond("Usage: /broadcast <message>");return true}
  featureMu.Lock();chats:=make([]int64,0,len(features.Chats));for cid,on:=range features.Chats{if on{chats=append(chats,cid)}};featureMu.Unlock()
  sent,failed:=0,0
  for _,cid:=range chats { if _,err:=b.SendMessage(ctx,&bot.SendMessageParams{ChatID:cid,Text:args});err!=nil{failed++;log.Printf("broadcast to %d: %v",cid,err)}else{sent++} }
  respond(fmt.Sprintf("📣 Broadcast complete. Sent: %d, failed: %d.",sent,failed));return true
 case "/maintenance":
  if id==0||(!isOwner(id)&&!s.isSudo(id)){respond("⛔ Owner or sudo access required.");return true}
  state:=strings.ToLower(strings.TrimSpace(args));if state!="enable"&&state!="disable"{respond("Usage: /maintenance enable|disable");return true}
  featureMu.Lock();features.Maintenance=state=="enable";err:=saveFeatureDataLocked();featureMu.Unlock();if err!=nil{respond("Could not save maintenance state: "+short(err.Error(),120))}else{respond("🛠️ Maintenance "+state+".")};return true
 case "/addsudo","/delsudo","/rmsudo","/sudolist","/listsudo","/sudoers":
  if command=="/sudolist"||command=="/listsudo"||command=="/sudoers" {featureMu.Lock();ids:=[]string{};for uid,on:=range features.Sudoers{if on{ids=append(ids,strconv.FormatInt(uid,10))}};featureMu.Unlock();if len(ids)==0{respond("Sudo list is empty.")}else{respond("🛡️ Sudo user IDs:\n"+strings.Join(ids,"\n"))};return true}
  if id==0||!isOwner(id){respond("⛔ Only OWNER_ID can manage sudo users.");return true}
  target,err:=strconv.ParseInt(strings.TrimSpace(args),10,64);if err!=nil||target<=0{respond("Usage: "+command+" <Telegram user ID>");return true}
  featureMu.Lock();if command=="/addsudo"{features.Sudoers[target]=true}else{delete(features.Sudoers,target)};err=saveFeatureDataLocked();featureMu.Unlock()
  if err!=nil{respond("Could not save sudo list: "+short(err.Error(),120))}else if command=="/addsudo"{respond(fmt.Sprintf("✅ %d added as sudo.",target))}else{respond(fmt.Sprintf("✅ %d removed from sudo.",target))};return true
 case "/block","/unblock","/blocked","/blockedusers","/blusers":
  if command=="/blocked"||command=="/blockedusers"||command=="/blusers"{featureMu.Lock();ids:=[]string{};for uid,on:=range features.BannedUsers{if on{ids=append(ids,strconv.FormatInt(uid,10))}};featureMu.Unlock();if len(ids)==0{respond("No blocked users.")}else{respond("⛔ Blocked user IDs:\n"+strings.Join(ids,"\n"))};return true}
  if id==0||(!isOwner(id)&&!s.isSudo(id)){respond("⛔ Owner/sudo access required.");return true}
  target,err:=strconv.ParseInt(strings.TrimSpace(args),10,64);if err!=nil||target<=0{respond("Usage: "+command+" <Telegram user ID>");return true}
  featureMu.Lock();if command=="/block"{features.BannedUsers[target]=true}else{delete(features.BannedUsers,target)};err=saveFeatureDataLocked();featureMu.Unlock()
  if err!=nil{respond("Could not save block list: "+short(err.Error(),120))}else if command=="/block"{respond(fmt.Sprintf("⛔ User %d blocked from this bot.",target))}else{respond(fmt.Sprintf("✅ User %d unblocked.",target))};return true
 case "/gban","/globalban","/ungban","/gbanlist","/gbannedusers":
  if command=="/gbanlist"||command=="/gbannedusers"{featureMu.Lock();ids:=[]string{};for uid,on:=range features.BannedUsers{if on{ids=append(ids,strconv.FormatInt(uid,10))}};featureMu.Unlock();if len(ids)==0{respond("No globally blocked users.")}else{respond("🌐 Global block list (bot-level):\n"+strings.Join(ids,"\n"))};return true}
  if id==0||(!isOwner(id)&&!s.isSudo(id)){respond("⛔ Owner/sudo access required.");return true}
  target,err:=strconv.ParseInt(strings.TrimSpace(args),10,64);if err!=nil||target<=0{respond("Usage: "+command+" <Telegram user ID>");return true}
  featureMu.Lock();if command=="/ungban"{delete(features.BannedUsers,target)}else{features.BannedUsers[target]=true};err=saveFeatureDataLocked();featureMu.Unlock()
  if err!=nil{respond("Could not save global block: "+short(err.Error(),120))}else if command=="/ungban"{respond(fmt.Sprintf("✅ User %d unblocked globally.",target))}else{respond(fmt.Sprintf("🌐 User %d globally blocked from using this bot.",target))};return true
 case "/pause","/cpause","/resume","/cresume","/seek","/cseek","/seekback","/cseekback":
  respond("ℹ️ This Go Bot API edition sends audio as a completed Telegram audio message. The original PyTgCalls live voice-chat pause/resume/seek controls require an MTProto user session and group-call engine, which this build does not yet include.");return true
 case "/reload","/clean":
  if id==0||(!isOwner(id)&&!s.isSudo(id)){respond("⛔ Owner/sudo access required.");return true}
  if command=="/clean"{respond("🧹 Temporary download cleanup is performed automatically after each request.")}else{respond("♻️ Go bot reload is managed by your process supervisor; restart the service to reload code.")};return true
 case "/activevc","/vc","/activevoice","/activev","/av","/activevideo","/ac","/activecalls":
  s.mu.Lock();var chats []string;for cid,running:=range s.active{if running{chats=append(chats,strconv.FormatInt(cid,10))}};s.mu.Unlock()
  if len(chats)==0{respond("No active audio-download workers.")}else{respond("🎧 Active download chats:\n"+strings.Join(chats,"\n"))};return true
 }
 return false
}
func allowedSpeed(v float64) bool {for _,x:=range []float64{0.5,0.75,1,1.25,1.5,2}{if math.Abs(v-x)<0.001{return true}};return false}
func (s *Service) adminChatLocked(chat,user int64) bool { s.mu.Lock(); defer s.mu.Unlock(); return s.admins[chat][user] }
func (s *Service) applyConfiguredSpeed(ctx context.Context,chat int64,path string)(string,error){
 featureMu.Lock();speed:=features.Speeds[chat];featureMu.Unlock();if speed==0||math.Abs(speed-1)<0.001{return path,nil}
 out:=strings.TrimSuffix(path,filepath.Ext(path))+fmt.Sprintf(".speed-%g",speed)+filepath.Ext(path)
 if _,err:=os.Stat(out);err==nil{return out,nil}
 cmd:=exec.CommandContext(ctx,envOr("FFMPEG_PATH","ffmpeg"),"-y","-i",path,"-vn","-filter:a",fmt.Sprintf("atempo=%g",speed),out)
 if output,err:=cmd.CombinedOutput();err!=nil{return "",fmt.Errorf("%w: %s",err,short(string(output),180))}
 return out,nil
}
func (s *Service) afterSuccessfulTrack(chat int64,item music.Item){
 featureMu.Lock();n:=features.Loops[chat];if n<=0{featureMu.Unlock();return};features.Loops[chat]=n-1;err:=saveFeatureDataLocked();featureMu.Unlock()
 if err!=nil{log.Printf("save loop count: %v",err);return}
 if err:=s.queues.Enqueue(chat,item);err!=nil{log.Printf("repeat enqueue: %v",err)}
}

func (s *Service) settingsText(chat int64) string {
 featureMu.Lock()
 lang:=features.Languages[chat];if lang==""{lang="en"}
 mode:=features.PlayModes[chat];if mode==""{mode="Everyone"}
 loop:=features.Loops[chat]
 speed:=features.Speeds[chat];if speed==0{speed=1}
 maintenance:=features.Maintenance
 featureMu.Unlock()
 return fmt.Sprintf("⚙️ Mikku Go Settings\nLanguage: %s\nPlay mode: %s\nRepeat remaining: %d\nAudio processing speed: %.2gx\nQueue limit: %d requests per chat\nDelivery: Telegram audio message\nSource: yt-dlp-supported URLs\nMaintenance: %t\n\nNote: live group-call controls and inline callback settings are not implemented in this Bot API build.",lang,mode,loop,speed,music.MaxQueueItems,maintenance)
}
