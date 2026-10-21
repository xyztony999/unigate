# 手机或平板作为现场主设备：可行性备忘

最新范围：ADR-039 将离线协调及主备 App 后置，首版在线浏览器检票。本备忘保留为后续研究，不是首版主设备实现任务。

日期：2026-10-20；状态：技术研究与建议，未选定实现技术栈，未做设备实测。

## 已确认与待确认

用户选择公网断开但现场局域网可用，并询问手机/平板主设备及 Wi-Fi/蓝牙。该网络基线记录于 ADR-006。后续用户已确认 Android 手机/平板主设备，接受专用 App 方向（ADR-007）；iPad 主设备不列为首版要求。Wi-Fi 优先仍为工程建议，具体实现框架待选，尚未进行设备实测。

## Wi-Fi 与移动主设备

Android 官方提供 Local-only hotspot，可供连接设备的应用互相通信，且该网络不提供互联网接入。由此可设计 Android 手机/平板运行本地服务和数据库，其他设备连同一 Wi-Fi；是否由主设备开热点或使用外部路由器，需按设备和入口规模实测决定。热点数量限制、网络隔离、覆盖、供电及休眠均需验证，不宣称任意设备无需 SIM 或任意热点都可用。

来源：[Android 本地热点](https://developer.android.com/develop/connectivity/wifi/localonlyhotspot)、[Android 无线互联](https://developer.android.com/develop/connectivity/wifi)。

Android 的前台服务受系统要求和启动限制约束，不能保证永不被终止。建议专用主设备在活动期间持续运行 App 并供电，离线数据本地持久化，通信中断有明确提示。

来源：[Android 前台服务](https://developer.android.com/develop/background-work/services/fgs)。

iPhone/iPad 的原生 Network 框架可监听入站连接，因此移动主设备并非只能是 Android。但普通 App 的后台执行有时间限制；需验证前台持续运行、系统权限及实际连接行为，不能把锁屏或切换应用后的持续服务当作默认保证。首版优先 Android 是工程建议，不是 Apple 完全不支持的结论。

来源：[Apple NWListener](https://developer.apple.com/documentation/network/nwlistener)、[Apple 后台执行](https://developer.apple.com/documentation/uikit/extending-your-app-s-background-execution-time)、[Apple 本地网络权限](https://developer.apple.com/documentation/technotes/tn3179-understanding-local-network-privacy)。

## 蓝牙

原生 App 可以另行设计蓝牙通信，但需验证协议、中心/外设角色、同时连接数、断线重连及跨系统兼容。不能把“设备支持蓝牙”直接等同于现有浏览器能组建手机间检票网络。

Web Bluetooth 主要面向 BLE 外设连接且浏览器支持有限，不能作为全浏览器、多平台主从网络的默认实现。首版建议 Wi-Fi；蓝牙可另作设备发现或配对候选，不在尚未确认时新增必选功能。

来源：[Web Bluetooth](https://developer.mozilla.org/en-US/docs/Web/API/Web_Bluetooth_API)。

## 浏览器检票端的约束

若客户端通过网页调用摄像头，必须满足安全上下文要求；其他设备访问主设备的普通 HTTP 局域网 IP 不是可直接依赖的摄像头方案。需选择可信 HTTPS 的现场部署与离线解析、具备原生能力的客户端，或针对仅扫码枪输入的客户端分别设计。安装 PWA 本身不解决证书与局域网访问约束。

来源：[getUserMedia 安全上下文要求](https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getUserMedia)。

## 拟议核销原则

现场主设备在本次会话中持有唯一核销权，客户端提交请求，由其原子检查组剩余名额、持久化、再返回结果。核销操作有独立幂等标识；云端恢复连接不自动形成第二个并行写入方。组票从 3 人先入 2 人后，其他入口应看到剩余 1 人。

这是建议的架构原则，不是已经完成的接口或协议规格。主设备故障时是否暂停、如何切换及如何处理最后一笔不确定提交，仍需明确；不默认自动切换为自由核销并容忍超额。

## 更新记录

- 2026-10-20：核实官方平台能力，提出 Android App 主设备与 Wi-Fi 优先方案，保留 iPad 及浏览器客户端约束。
- 2026-10-20：用户确认 Android 手机/平板主设备与专用 App 方向；更新确认边界，不改变尚未实测的技术结论。
