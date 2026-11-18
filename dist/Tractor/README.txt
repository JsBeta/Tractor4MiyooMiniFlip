拖拉机 · 升级
==============

四人两副牌的“拖拉机 / 升级 / 双扣”，为 Miyoo Mini Flip 重新设计的界面。
原生 Go 编写，直接写 /dev/fb0，不依赖 OnionOS，也不需要 SDL。

安装
----
1. 把整个 Tractor 文件夹复制到 SD 卡：
      /mnt/SDCARD/App/Tractor/
   里面应有：launch.sh、icon.png、tractor-arm、tractor-arm64、本说明。
2. 安全弹出 SD 卡，插回机器，开机进入 App 菜单，运行“拖拉机”。

操作
----
  十字键      选牌 / 移动光标
  A           选中 / 取消一张牌
  X           出牌 / 确认 / 亮主 / 扣底
  Y           提示（自动选出一手合法牌）
  B           取消选择 / 不亮
  START       暂停菜单（保存进度）
  SELECT      玩法说明
  L1 / L2     出牌日志（↑↓ 滚动、←→ 翻页，再按 L 或 B 返回）

玩法
----
· 两副牌 108 张，四人两两结对，每人 25 张，底牌 8 张。
· 庄家一方守分，闲家一方抢分：5 记 5 分，10 和 K 各记 10 分，共 200 分。
· 闲家拿到 80 分即可上台；不足 80 分庄家一方升级：
    - 闲家 0 分：大光，升 3 级
    - 闲家 40 分以下：小光，升 2 级
    - 闲家 80 分以下：升 1 级
    - 闲家每多 40 分，上台后多升 1 级
· 主牌：大小王、所有级牌、主花色。
  单张大小：大王 > 小王 > 主级牌 > 副级牌 > 主花色 A K Q J 10 … 2。
· 牌型：单张、对子、拖拉机（同花色相邻两对）、甩牌。
· 跟牌：先跟同花色；有对必须出对，有拖拉机必须出拖拉机；
  没有该花色时才能垫牌或用主牌毙牌。
· 亮主：拿到级牌即可亮主，一对级牌可以反主；无人亮主则打无主。
· 扣底：庄家拿起底牌 8 张，再扣回 8 张。
  闲家赢下最后一墩时底牌分翻倍：单张 ×2，对子 ×4，拖拉机 ×8；
  甩牌按其结构：含拖拉机 ×8，含对子 ×4，否则 ×2。

保存进度
--------
· 发牌、亮主、扣底以及每墩结束后自动保存。
· 暂停菜单选“保存进度”可手动保存。
· 下次启动时标题界面选“继续上次牌局”。
· 存档位置：/mnt/SDCARD/Saves/Tractor/save.json

故障排查
--------
· 画面上下颠倒：
    在 /mnt/SDCARD/Saves/Tractor/ 里新建一个空文件，名字叫 rotate0，
    重新启动游戏即可。
· 想检查屏幕方向和按键：
    在 /mnt/SDCARD/Saves/Tractor/ 里新建空文件 diag，重新启动，
    会显示彩色条和按键测试；按 SELECT 退出。检查完删掉 diag。
· 按键没反应：
    看 /mnt/SDCARD/Saves/Tractor/keys.log，里面记录了收到的键码。
· 启动后黑屏：
    看 /mnt/SDCARD/Saves/Tractor/launcher.log，里面有屏幕参数和错误信息。
· 屏幕分辨率必须为 752×560；不一致时程序会退出并写入日志。

构建
----
  GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" ./cmd/tractor
  GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" ./cmd/tractor

源码与构建说明见项目根目录的 README.md。
