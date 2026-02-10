#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
华为呼叫会议调试脚本 - 打印详细响应
"""

import argparse
import json
import logging
import subprocess
from typing import Dict, Any, Optional

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def execute_curl_command(endpoint: str, data: Optional[Dict[str, Any]] = None,
                         additional_headers: Optional[Dict[str, str]] = None) -> Optional[str]:
    """执行curl命令并返回原始响应"""
    try:
        cmd = [
            "curl", "-k", "-X", "POST",
            "-H", "Content-Type: application/json",
            "-H", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
            "--tlsv1.0", "--tls-max", "1.2",
            "--connect-timeout", "30",
        ]

        if additional_headers:
            for key, value in additional_headers.items():
                cmd.extend(["-H", f"{key}: {value}"])

        if data:
            cmd.extend(["-d", json.dumps(data, ensure_ascii=False)])

        cmd.append(f"https://10.62.10.3/action.cgi?ActionID={endpoint}")

        print(f"\n执行curl命令:")
        print(f"  端点: {endpoint}")
        print(f"  数据: {json.dumps(data, indent=2, ensure_ascii=False) if data else '(无)'}")
        print(f"  完整命令: {' '.join(cmd)}\n")

        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)

        if result.returncode == 0:
            print(f"原始响应:")
            print(f"  {result.stdout}\n")

            try:
                response = json.loads(result.stdout)
                print(f"解析后的JSON:")
                print(f"  {json.dumps(response, indent=2, ensure_ascii=False)}\n")
                return response
            except json.JSONDecodeError:
                print(f"响应不是JSON格式")
                return {"raw_response": result.stdout}
        else:
            print(f"curl命令失败: {result.stderr}")
            return None

    except Exception as e:
        print(f"curl请求异常: {e}")
        return None


def main():
    print("=" * 60)
    print("华为呼叫会议调试")
    print("=" * 60)

    # 步骤1: 获取会话ID
    print("\n步骤1: 获取会话ID")
    response = execute_curl_command("Web_RequestSessionID")
    if response and response.get("success") == 1:
        data = json.loads(response["data"])
        session_id = data.get("acSessionId") or response.get("acSessionId")
        if not session_id:
            # 从响应中提取（可能data字段是空的）
            print("  从data字段获取acSessionId失败")
            return
        print(f"  会话ID: {session_id}")
    else:
        print("  获取会话ID失败")
        return

    # 步骤2: 认证
    print("\n步骤2: 认证")
    auth_data = {
        "user": "api",
        "password": "Hubei@1992"
    }
    response = execute_curl_command(
        "WEB_RequestCertificateAPI",
        auth_data,
        additional_headers={"Cookie": f"SessionID={session_id}"}
    )
    if response and response.get("success") == 1:
        print("  认证成功")
    else:
        print("  认证失败")
        return

    # 步骤3: 替换会话ID
    print("\n步骤3: 替换会话ID")
    response = execute_curl_command(
        "WEB_ChangeSessionID",
        additional_headers={"Cookie": f"SessionID={session_id}"}
    )
    if response and response.get("success") == 1:
        data = json.loads(response["data"])
        new_session_id = data.get("acSessionId")
        if new_session_id:
            session_id = new_session_id
        print(f"  新会话ID: {session_id}")
    else:
        print("  替换会话ID失败")
        return

    # 步骤4: 呼叫会议 - 测试两种API名称
    call_data = {
        "bIsLdapCall": 0,
        "bIsVideoCall": 0,
        "ucEnableH239": 1,
        "stSiteInfo": {
            "uwID": 0,
            "szName": "521270003",
            "szPName": "",
            "ucType": 8,
            "bIsLdap": 0,
            "ucDevice": 0,
            "ucOnline": 0,
            "uwSortPos": 0,
            "stSIP": {
                "ucBaudRate": 1920,
                "szAlias": "",
                "szIP": "",
                "szUri": ""
            }
        }
    }

    print("\n步骤4a: 呼叫会议 (使用 WEB_CallSiteAPI - 带API后缀)")
    response = execute_curl_command(
        "WEB_CallSiteAPI",
        call_data,
        additional_headers={"Cookie": f"SessionID={session_id}"}
    )

    print("\n" + "=" * 30)
    if response and response.get("success") == 1:
        print("WEB_CallSiteAPI: 成功!")
    else:
        print(f"WEB_CallSiteAPI: 失败")

    print("\n步骤4b: 呼叫会议 (使用 WEB_CallSite - 不带API后缀)")
    response = execute_curl_command(
        "WEB_CallSite",
        call_data,
        additional_headers={"Cookie": f"SessionID={session_id}"}
    )

    print("\n" + "=" * 60)
    if response:
        success = response.get("success")
        if success == 1:
            print("WEB_CallSite: 成功!")
        else:
            print(f"WEB_CallSite: 失败 (success={success})")
            if "exception" in response:
                print(f"  exception: {response['exception']}")
            if "error" in response:
                print(f"  error: {response['error']}")
    else:
        print("WEB_CallSite: 无响应")
    print("=" * 60)


if __name__ == '__main__':
    main()
