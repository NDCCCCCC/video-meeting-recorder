#!/usr/bin/env python3
"""华为终端认证测试脚本 - 兼容老版本TLS"""

import requests
import json
import ssl
import urllib3
from requests.adapters import HTTPAdapter
from urllib3.util.ssl_ import create_urllib3_context

# 禁用SSL警告
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# 配置
BASE_URL = "https://10.62.10.3:443"
USERNAME = "api"
PASSWORD = "Hubei@1992"


class TLSAdapter(HTTPAdapter):
    """自定义TLS适配器，支持老版本的TLS配置"""

    def init_poolmanager(self, *args, **kwargs):
        # 创建支持老版本TLS的SSL上下文
        context = create_urllib3_context()

        # 设置TLS 1.2（华为TE40支持的版本）
        context.minimum_version = ssl.TLSVersion.TLSv1_2
        context.maximum_version = ssl.TLSVersion.TLSv1_2

        # 设置华为终端支持的密码套件
        # 使用OpenSSL格式的密码套件名称
        context.set_ciphers('HIGH:!aNULL:!eNULL:!MD5')

        # 不验证主机名（兼容自签名证书）
        context.check_hostname = False
        context.verify_mode = ssl.CERT_NONE

        kwargs['ssl_context'] = context
        return super().init_poolmanager(*args, **kwargs)


def mask_string(s):
    """隐藏敏感信息"""
    if len(s) <= 8:
        return "****"
    return s[:4] + "****" + s[-4:]


def test_huawei_auth():
    print("=== 华为终端认证流程测试 ===")
    print(f"目标: {BASE_URL}")
    print(f"用户名: {USERNAME}\n")

    # 创建session，使用自定义TLS适配器
    session = requests.Session()
    session.verify = False
    session.mount('https://', TLSAdapter())

    try:
        # 步骤1: 获取会话ID
        print("步骤1: 获取会话ID (Web_RequestSessionID)")
        session_url = f"{BASE_URL}/action.cgi?ActionID=Web_RequestSessionID"
        headers = {"userType": "web"}

        resp1 = session.post(session_url, headers=headers)
        print(f"  响应状态: {resp1.status_code}")
        print(f"  响应内容: {resp1.text}")

        data1 = resp1.json()
        if data1.get("success") != 1:
            print("  获取会话ID失败")
            return

        # 从Cookie中获取SessionID
        session_id = None
        for cookie in session.cookies:
            if cookie.name == "SessionID":
                session_id = cookie.value
                print(f"  从Cookie获取会话ID: {mask_string(session_id)}")
                break

        if not session_id:
            print("  错误: 未能获取到会话ID")
            return

        print(f"  最终会话ID: {mask_string(session_id)}\n")

        # 步骤2: 用户认证
        print("步骤2: 用户认证 (WEB_RequestCertificateAPI)")
        auth_url = f"{BASE_URL}/action.cgi?ActionID=WEB_RequestCertificateAPI"

        # 尝试不同的字段名格式
        auth_success = False
        for field_format in [("大写", "User", "Password"), ("小写", "user", "password")]:
            print(f"  尝试 {field_format[0]} 字段名:")

            auth_headers = {
                "userType": "web",
                "Content-Type": "application/json",
                "Cookie": f"SessionID={session_id}"
            }

            auth_payload = {
                field_format[1]: USERNAME,
                field_format[2]: PASSWORD
            }

            resp2 = session.post(auth_url, headers=auth_headers, json=auth_payload)
            print(f"    响应状态: {resp2.status_code}")
            print(f"    响应内容: {resp2.text[:200]}")

            data2 = resp2.json()
            if data2.get("success") == 1:
                print(f"    *** 认证成功! (使用{field_format[0]}字段名) ***\n")
                auth_success = True
                break
            else:
                error_id = data2.get("error", {}).get("id") or data2.get("exception", {}).get("id")
                print(f"    认证失败，错误码: {error_id}")

        print()

        # 步骤3: 替换会话ID
        if auth_success:
            print("步骤3: 替换会话ID (WEB_ChangeSessionID)")
            change_url = f"{BASE_URL}/action.cgi?ActionID=WEB_ChangeSessionID"

            change_headers = {
                "userType": "web",
                "Cookie": f"SessionID={session_id}"
            }

            resp3 = session.post(change_url, headers=change_headers)
            print(f"  响应状态: {resp3.status_code}")
            print(f"  响应内容: {resp3.text}")

            data3 = resp3.json()
            if data3.get("success") == 1:
                print("  *** 替换会话ID成功! ***")
            else:
                error_id = data3.get("error", {}).get("id")
                print(f"  替换失败，错误码: {error_id}")

    except Exception as e:
        print(f"错误: {e}")
        import traceback
        traceback.print_exc()

    print("\n=== 测试完成 ===")


if __name__ == "__main__":
    test_huawei_auth()
