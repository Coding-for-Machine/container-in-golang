<p align="center">
  <a href="https://github.com/Coding-for-Machine/container-in-golang">
    <img src="./images/image_linux.png" width="30%" style="border-radius:25px;" />
  </a>
</p>

```bash
man 7 namespaces
```

| Namespace | Flag            | Isolates                                     |
| --------- | --------------- | -------------------------------------------- |
| Cgroup    | CLONE_NEWCGROUP | Cgroup root katalogi (cgroup iyerarxiyasi)   |
| IPC       | CLONE_NEWIPC    | System V IPC va POSIX message queue’lar      |
| Network   | CLONE_NEWNET    | Tarmoq interfeyslari, stack, portlar va h.k. |
| Mount     | CLONE_NEWNS     | Mount point’lar (fayl tizimlari xaritasi)    |
| PID       | CLONE_NEWPID    | Process ID bo‘shlig‘i                        |
| Time      | CLONE_NEWTIME   | Boot va monotonic soatlar                    |
| User      | CLONE_NEWUSER   | User va group ID’lar                         |
| UTS       | CLONE_NEWUTS    | Hostname va NIS domain nomi                  |


### 1. PID namespace (processlar)

**Nima qiladi?**  
PID namespace jarayonlar (processlar) ID’larini ajratadi: har bir PID namespace ichida processlar o‘zlariga xos PID bilan 1, 2, 3… dan boshlab ko‘rinadi, lekin bu PIDlar tashqi tizimdagi PIDlardan farq qiladi.

**Oddiy misol**  
Asl tizimda, masalan:

- `ps` → PID 1: `systemd`, PID 1234: `bash`, va hokazo.

Yangi PID namespace ichida esa go‘yoki kichik “mini-tizim” bor:

- Shu namespace ichida ishga tushgan `bash` PID 1 bo‘lib ko‘rinadi, ya’ni o‘z ichida “init” process vazifasini bajaradi.

**Amaliy misol (konseptual)**  

Yangi PID namespace’da shell ishga tushirish:

```bash
unshare -p -f --mount-proc bash
```

Shundan so‘ng yangi shell ichida:

- `ps` desangiz, `bash` PID 1 bo‘lib ko‘rinadi.
- Tashqaridagi tizimda esa xuddi shu `bash` boshqa, katta PID bilan (masalan, 5000) ko‘rinadi.

Bu PID namespace konteyner ichida “faqat o‘zimning processlarim bor” degan illyuziyani yaratadi va boshqa namespace’lar bilan birga ishlaganda to‘liq ajratilgan muhit hosil qiladi.

### 2. Network namespace (tarmoq)

**Nima qiladi?**  
Network namespace har bir namespace uchun to‘liq **alohida tarmoq stack** beradi:  
- O‘z `lo` (loopback) interfeysi  
- O‘z `eth0`, `ip addr`, `ip route` jadvallari  
- O‘z `iptables` qoidalari va firewall sozlamalari  

**Oddiy misol**  
**Asl tizimda:**
```bash
ip addr
```
Natija: `eth0`, `lo`, balki `wlan0` interfeyslari ko‘rinadi.

**Yangi network namespace ichida:**  
Faqat `lo` interfeysi bo‘ladi, u ham `DOWN` holatda, hech qanday IP manzil yo‘q.

**Amaliy misol (konseptual)**  

1. **Network namespace yaratish:**
```bash
ip netns add ns1
```

2. **Ichida tekshirish:**
```bash
ip netns exec ns1 ip addr
```
Natija: Faqat `lo` (DOWN holatda).

3. **Virtual ethernet juftligi yaratib, ulash:**
```bash
# veth juftligi yaratish
ip link add veth0 type veth peer name veth1

# Bir uchini ns1 ga ko'chirish
ip link set veth1 netns ns1

# Tashqariga IP berish
ip addr add 192.168.1.1/24 dev veth0
ip link set veth0 up

# ns1 ichiga IP berish
ip netns exec ns1 ip addr add 192.168.1.2/24 dev veth1
ip netns exec ns1 ip link set veth1 up
ip netns exec ns1 ip link set lo up
```

**Endi `ns1` ichida alohida tarmoq dunyosi hosil bo‘ldi:**
- `ip netns exec ns1 ip addr` → `lo` + `veth1` (192.168.1.2)
- `ping 192.168.1.1` → ishlaydi (tashqi veth0 bilan aloqa)
- Tashqaridagi tizimda `ns1` ning tarmoq interfeyslari ko‘rinmaydi!

Bu container’ning o‘z `eth0`si, o‘z IP’lari, o‘z tarmoq sozlamalari bo‘lishi uchun asosiy mexanizm.

### 3. Mount namespace (fayl tizimlari)

**Nima qiladi?**  
Mount namespace qaysi fayl tizimi qayerga **mount** qilinganini ajratadi. Har bir namespace ichida o‘ziga xos mount xaritasi (fayl tizimlari ko‘rinishi) bo‘ladi.

**Oddiy misol**  
**Asl tizimda:**  
`/home`, `/var`, `/mnt/data` va boshqa mount point’lar bor.

**Yangi mount namespace ichida:**  
`/mnt/test` ga boshqa disk yoki `tmpfs` mount qilishingiz mumkin. Bu faqat shu namespace ichida ko‘rinadi, tashqaridagi tizim bundan bexabar qoladi.

**Amaliy misol**  
```bash
unshare -m bash
```

Yangi shell ichida:  
```bash
mount -t tmpfs tmpfs /mnt/tmp
ls /mnt/
```
Natija: `/mnt/tmp` paydo bo‘ladi.

**Tashqaridagi shell’da:**  
```bash
ls /mnt/
```
Natija: `/mnt/tmp` ko‘rinmaydi!

Bu konteyner ichida **alohida root filesystem** yaratish uchun asosiy mexanizm.

***

### 4. UTS namespace (hostname)

**Nima qiladi?**  
Hostname va NIS domain nomini ajratadi. Har bir namespace o‘zining hostname’iga ega bo‘lishi mumkin.

**Oddiy misol**  
**Asl tizim:** `hostname` → `server-main`

**Yangi UTS namespace ichida:**  
```bash
hostname container1
```
Faqat shu namespace ichida `container1` bo‘ladi, tashqarida esa `server-main` qoladi.

**Amaliy misol**  
```bash
unshare -u bash
```

Ichida:  
```bash
hostname          # server-main
hostname mycontainer
hostname          # mycontainer
```

**Tashqaridagi shell’da:**  
```bash
hostname          # server-main (o‘zgarmagan)
```

***

### 5. User namespace (user va group ID)

**Nima qiladi?**  
User va group ID’larni ajratadi. Eng muhim xususiyati: container ichida **UID 0 (root)** bo‘lib, tashqarida oddiy user bo‘lib qolish mumkin.

**Oddiy misol**  
Siz `asadbek` (UID 1000) bo‘lsangiz:  
**User namespace ichida:** `id` → `uid=0(root) gid=0(root)`  
**Tashqarida:** hali ham UID 1000 (haqiqiy root emassiz).

**Amaliy misol**  
```bash
unshare -U bash
```

Ichida:  
```bash
id                # uid=0(root) ko'rinishi mumkin
cat /proc/self/uid_map  # UID mapping ko'rsatadi
```

`/proc/self/uid_map` va `/gid_map` orqali ichki UID’larni tashqi UID’lar bilan moslashtirish mumkin. Bu **rootless container**lar uchun asosiy mexanizm.

***

### 6. IPC namespace (inter-process communication)

**Nima qiladi?**  
System V IPC va POSIX message queue resurslarini ajratadi:  
- Shared memory segmentlar  
- Semaforlar  
- Message queue’lar  

**Oddiy misol**  
Asl tizimda `shmget`, `semget` orqali umumiy IPC resurslar ishlatiladi.  
Yangi IPC namespace ichida yaratilgan shared memory faqat shu namespace’da ko‘rinadi.

**Amaliy misol**  
```bash
unshare -i bash
```

Ichida:  
```bash
ipcs -m          # Shared memory (bo'sh)
```
Yangi shared memory yaratsangiz, faqat shu namespace ichida `ipcs` da ko‘rinadi.

***

### 7. Cgroup namespace

**Nima qiladi?**  
Process qaysi cgroup’da ekanligini ajratadi. Container ichida `/proc/self/cgroup` o‘ziga xos “ildiz” cgroup’ni ko‘radi.

**Oddiy misol**  
**Asl tizim:** `/sys/fs/cgroup/...` murakkab ierarxiya  
**Cgroup namespace ichida:** Container o‘zini cgroup ildizida deb ko‘radi.

**Amaliy misol**  
```bash
cat /proc/self/cgroup  # Tashqarida: /docker/xxx/main
```
Cgroup namespace ichida esa: `/` (ildiz) ko‘rinadi.

Bu `top`, `systemd` kabi vositalar container ichida “faqat o‘z cgroup’larim” deb ishlashi uchun kerak.

***

### 8. Time namespace

**Nima qiladi?**  
Boot time va monotonic clock’larni ajratadi. Har xil “vaqt” ko‘rinishlari berish mumkin.

**Oddiy misol**  
**Asl tizim:** `clock_gettime(CLOCK_MONOTONIC)` barcha uchun bir xil  
**Time namespace ichida:** Offset qo‘yib “10 daqiqa oldin ishga tushgandek” ko‘rsatish.

**Amaliy misol**  
```bash
unshare -T bash
cat /proc/uptime  # Tashqaridagidan farqli uptime
```

Test va debugging uchun foydali.

***

## Yakuniy tushuncha

Har bir namespace turi **bitta resurs turini** izolyatsiya qiladi:

| Namespace | Izolyatsiya qiladigan resurs |
|:----------|:-----------------------------|
| **PID** | Process ID bo‘shlig‘i |
| **NET** | Tarmoq stack interfeyslar |
| **MNT** | Mount point xaritasi |
| **UTS** | Hostname |
| **USER** | UID/GID mapping |
| **IPC** | Shared memory, semaforlar |
| **CGROUP** | Cgroup ierarxiyasi |
| **TIME** | Vaqt soatlari |

**Containerlar** (Docker, Podman) odatda **PID + NET + MNT + UTS + USER** ni birlashtirib, to‘liq “mini-Linux” illyuziyasini yaratadi.