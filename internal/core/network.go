package core

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

// WakeOnLAN 发送网络唤醒魔术包
func WakeOnLAN(mac, netmask, deviceIP string, port int, iface string) error {
	// 解析MAC地址
	macAddr, err := parseMAC(mac)
	if err != nil {
		return fmt.Errorf("无效的MAC地址: %w", err)
	}

	// 构造魔术包
	packet := buildMagicPacket(macAddr)

	// 计算广播地址
	destAddr, err := getBroadcastAddr(deviceIP, netmask, port)
	if err != nil {
		return fmt.Errorf("无法计算广播地址: %w", err)
	}

	// 获取网络接口
	conn, err := getConnection(iface)
	if err != nil {
		return fmt.Errorf("无法创建连接: %w", err)
	}
	defer conn.Close()

	// 发送魔术包
	_, err = conn.WriteTo(packet, destAddr)
	if err != nil {
		return fmt.Errorf("发送魔术包失败: %w", err)
	}

	return nil
}

// parseMAC 解析MAC地址
func parseMAC(mac string) ([]byte, error) {
	// 支持 XX:XX:XX:XX:XX:XX 和 XX-XX-XX-XX-XX-XX 格式
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")

	if len(mac) != 12 {
		return nil, fmt.Errorf("MAC地址长度不正确")
	}

	macAddr := make([]byte, 6)
	for i := 0; i < 6; i++ {
		var b byte
		_, err := fmt.Sscanf(mac[i*2:i*2+2], "%02x", &b)
		if err != nil {
			return nil, err
		}
		macAddr[i] = b
	}

	return macAddr, nil
}

// buildMagicPacket 构造魔术包
func buildMagicPacket(mac []byte) []byte {
	// 魔术包格式: 6字节0xFF + 16次重复的MAC地址
	packet := make([]byte, 102)

	// 填充前6字节为0xFF
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}

	// 重复16次MAC地址
	for i := 1; i <= 16; i++ {
		copy(packet[i*6:], mac)
	}

	return packet
}

// getBroadcastAddr 根据设备IP和子网掩码计算广播地址
func getBroadcastAddr(deviceIP, netmask string, port int) (*net.UDPAddr, error) {
	// 如果未指定掩码，使用全局广播
	if netmask == "" || netmask == "255.255.255.255" {
		return net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", port))
	}

	// 解析设备IP
	ip := net.ParseIP(deviceIP)
	if ip == nil {
		return nil, fmt.Errorf("无效的设备IP地址: %s", deviceIP)
	}
	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("仅支持IPv4地址")
	}

	// 解析子网掩码
	mask := net.ParseIP(netmask)
	if mask == nil {
		return nil, fmt.Errorf("无效的子网掩码: %s", netmask)
	}
	mask = mask.To4()
	if mask == nil {
		return nil, fmt.Errorf("子网掩码必须是IPv4格式")
	}

	// 计算广播地址: IP | (~Mask)
	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcast.String(), port))
}

// getConnection 获取UDP连接
func getConnection(iface string) (*net.UDPConn, error) {
	var localAddr *net.UDPAddr

	if iface != "" && iface != "default" {
		localAddr = &net.UDPAddr{IP: net.ParseIP(iface)}
	}

	conn, err := net.DialUDP("udp", localAddr, &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 0,
	})
	if err != nil {
		return nil, err
	}

	// 设置广播选项
	if err := conn.SetWriteBuffer(1024); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// Ping 检测设备是否在线
func Ping(ip string, timeout time.Duration) (bool, error) {
	// 使用TCP连接测试（更可靠）
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "445"), timeout)
	if err != nil {
		// 尝试其他常见端口
		conn, err = net.DialTimeout("tcp", net.JoinHostPort(ip, "139"), timeout)
		if err != nil {
			// 使用ICMP ping (需要root权限)
			return icmpPing(ip, timeout)
		}
	}

	if conn != nil {
		conn.Close()
		return true, nil
	}

	return false, nil
}

// icmpPing 使用ICMP协议ping
func icmpPing(ip string, timeout time.Duration) (bool, error) {
	// 解析IP地址
	addr, err := net.ResolveIPAddr("ip4", ip)
	if err != nil {
		return false, err
	}

	// 创建ICMP连接
	conn, err := net.DialIP("ip4:icmp", nil, addr)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// 设置超时
	conn.SetDeadline(time.Now().Add(timeout))

	// 构造ICMP Echo请求
	msg := make([]byte, 8)
	msg[0] = 8  // Echo Request
	msg[1] = 0  // Code
	msg[2] = 0  // Checksum (待计算)
	msg[3] = 0  // Checksum
	msg[4] = 0  // Identifier
	msg[5] = 13 // Identifier
	msg[6] = 0  // Sequence
	msg[7] = 37 // Sequence

	// 计算校验和
	checksum := calcChecksum(msg)
	msg[2] = byte(checksum >> 8)
	msg[3] = byte(checksum & 0xff)

	// 发送请求
	if _, err := conn.Write(msg); err != nil {
		return false, err
	}

	// 接收响应
	reply := make([]byte, 1500)
	n, err := conn.Read(reply)
	if err != nil {
		return false, err
	}

	// 检查响应
	if n > 0 && reply[0] == 0 { // Echo Reply
		return true, nil
	}

	return false, nil
}

// calcChecksum 计算校验和
func calcChecksum(msg []byte) uint16 {
	sum := uint32(0)
	for i := 0; i < len(msg)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(msg[i : i+2]))
	}
	if len(msg)%2 == 1 {
		sum += uint32(msg[len(msg)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	return ^uint16(sum)
}
