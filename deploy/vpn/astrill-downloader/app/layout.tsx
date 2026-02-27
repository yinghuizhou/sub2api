import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Astrill OpenVPN 批量下载器',
  description: '自动遍历服务器，批量下载 .ovpn 配置文件',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body className="antialiased">{children}</body>
    </html>
  );
}
