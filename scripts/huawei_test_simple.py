#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
华为视频会议API测试脚本 - 简化版（避免Unicode编码问题）
"""

import argparse
import json
import logging
import subprocess
import time
from typing import Dict, Any, Optional, List
from dataclasses import dataclass

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@dataclass
class HuaweiConfig:
    """华为会议系统配置"""
    server: str
    username: str
    password: str
    timeout: int = 30
    tls_version: str = "1.0"


class HuaweiAPITester:
    """华为视频会议API测试器"""

    def __init__(self, config: HuaweiConfig):
        self.config = config
        self.session_id: Optional[str] = None
        self.conference_number: Optional[str] = None

    def execute_curl_command(self, endpoint: str, data: Optional[Dict[str, Any]] = None,
                           content_type: str = "application/json", method: str = "POST",
                           additional_headers: Optional[Dict[str, str]] = None) -> Optional[Dict[str, Any]]:
        """执行curl命令"""
        try:
            # 构建curl命令
            cmd = [
                "curl", "-k", "-X", method,
                "-H", f"Content-Type: {content_type}",
                "-H", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                "--tlsv1.0", "--tls-max", "1.2",
                "--connect-timeout", str(self.config.timeout),
            ]

            # 添加额外的请求头
            if additional_headers:
                for key, value in additional_headers.items():
                    cmd.extend(["-H", f"{key}: {value}"])

            # 添加数据
            if data:
                if content_type == "application/json":
                    cmd.extend(["-d", json.dumps(data)])
                else:
                    # 表单格式
                    form_data = "&".join([f"{k}={v}" for k, v in data.items()])
                    cmd.extend(["-d", form_data])

            # 添加端点
            cmd.append(f"https://{self.config.server}/action.cgi?ActionID={endpoint}")

            logger.info(f"执行curl命令: {' '.join(cmd)}")

            # 执行curl命令
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=self.config.timeout)

            if result.returncode == 0:
                try:
                    response = json.loads(result.stdout)
                    logger.info(f"curl请求成功: {endpoint}")
                    return response
                except json.JSONDecodeError:
                    logger.warning(f"响应不是JSON格式: {result.stdout}")
                    return {"raw_response": result.stdout}
            else:
                logger.error(f"curl命令失败: {result.stderr}")
                return None

        except subprocess.TimeoutExpired:
            logger.error(f"curl请求超时: {endpoint}")
            return None
        except Exception as e:
            logger.error(f"curl请求异常: {e}")
            return None

    def get_session_id(self) -> bool:
        """获取会话ID"""
        logger.info("正在获取会话ID...")
        response = self.execute_curl_command("Web_RequestSessionID")

        if response and response.get("success") == 1:
            try:
                data = json.loads(response["data"])
                self.session_id = data.get("acSessionId")
                logger.info(f"获取到会话ID: {self.session_id}")
                return True
            except (json.JSONDecodeError, KeyError) as e:
                logger.error(f"解析会话ID失败: {e}")
                return False

        logger.error("获取会话ID失败")
        return False

    def authenticate(self) -> bool:
        """认证"""
        if not self.session_id:
            logger.error("没有可用的会话ID")
            return False

        logger.info("正在认证...")

        # 认证请求体
        auth_data = {
            "user": self.config.username,
            "password": self.config.password
        }

        # 尝试不同的认证端点
        endpoints = ["WEB_RequestCertificateAPI", "WEB_Login"]

        for endpoint in endpoints:
            logger.info(f"尝试使用端点: {endpoint}")
            response = self.execute_curl_command(
                endpoint,
                auth_data,
                content_type="application/json",
                additional_headers={"Cookie": f"SessionID={self.session_id}"}
            )

            if response:
                if response.get("success") == 1:
                    logger.info(f"{endpoint} 认证成功")
                    return True
                else:
                    error_info = response.get("exception", {})
                    error_id = error_info.get("id") if isinstance(error_info, dict) else None
                    logger.warning(f"{endpoint} 认证失败，错误代码: {error_id}")
                    if error_id == 100666767:
                        logger.info("登录成功，但未进入会议（错误代码100666767）")
                        return True

        logger.error("认证失败")
        return False

    def change_session_id(self) -> bool:
        """替换会话ID"""
        if not self.session_id:
            logger.error("没有可用的会话ID")
            return False

        logger.info("正在替换会话ID...")
        response = self.execute_curl_command(
            "WEB_ChangeSessionID",
            additional_headers={"Cookie": f"SessionID={self.session_id}"}
        )

        if response and response.get("success") == 1:
            logger.info("会话ID替换成功")
            try:
                data = json.loads(response["data"])
                self.session_id = data.get("acSessionId")
                logger.info(f"新的会话ID: {self.session_id}")
                return True
            except (json.JSONDecodeError, KeyError) as e:
                logger.error(f"解析新会话ID失败: {e}")
                return False

        logger.error("会话ID替换失败")
        return False

    def call_conference(self, conference_number: str) -> bool:
        """呼叫会议"""
        if not self.session_id:
            logger.error("没有可用的会话ID，请先登录")
            return False

        logger.info(f"正在呼叫会议: {conference_number}")

        call_data = {
            "bIsLdapCall": 0,
            "bIsVideoCall": 0,
            "ucEnableH239": 1,
            "stSiteInfo": {
                "uwID": 0,
                "szName": conference_number,
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

        response = self.execute_curl_command(
            "WEB_CallSite",
            call_data,
            additional_headers={"Cookie": f"SessionID={self.session_id}"}
        )

        if response and response.get("success") == 1:
            logger.info(f"会议呼叫成功: {conference_number}")
            self.conference_number = conference_number
            return True
        else:
            error_info = response.get("exception", {}) if response else {}
            error_id = error_info.get("id") if isinstance(error_info, dict) else None
            logger.error(f"会议呼叫失败，错误代码: {error_id}")
            return False

    def run_test(self) -> bool:
        """运行完整的API测试"""
        logger.info("开始华为视频会议API测试...")

        try:
            # 1. 获取会话ID
            if not self.get_session_id():
                logger.error("测试失败: 无法获取会话ID")
                return False

            # 2. 认证
            if not self.authenticate():
                logger.error("测试失败: 认证失败")
                return False

            # 3. 替换会话ID
            if not self.change_session_id():
                logger.error("测试失败: 替换会话ID失败")
                return False

            logger.info("华为视频会议API测试完成 - 所有核心功能正常")
            return True

        except Exception as e:
            logger.error(f"测试过程中发生异常: {e}")
            return False


def main():
    """主函数"""
    parser = argparse.ArgumentParser(description='华为视频会议API测试工具')
    parser.add_argument('--server', required=True, help='华为视频会议服务器地址')
    parser.add_argument('--username', required=True, help='用户名')
    parser.add_argument('--password', required=True, help='密码')
    parser.add_argument('--timeout', type=int, default=30, help='请求超时时间（秒）')
    parser.add_argument('--conference', help='会议号码（用于呼叫测试）')

    args = parser.parse_args()

    # 创建配置
    config = HuaweiConfig(
        server=args.server,
        username=args.username,
        password=args.password,
        timeout=args.timeout
    )

    # 创建测试器
    tester = HuaweiAPITester(config)

    print("=== 华为视频会议API测试工具 ===")
    print(f"服务器: {config.server}")
    print(f"用户名: {config.username}")
    if args.conference:
        print(f"测试会议: {args.conference}")
    print("=" * 40)

    # 运行测试
    start_time = time.time()

    success = tester.run_test()

    # 如果提供了会议号码，测试呼叫功能
    if success and args.conference:
        print(f"\n测试呼叫会议: {args.conference}")
        if tester.call_conference(args.conference):
            print("呼叫测试成功")
        else:
            print("呼叫测试失败")

    end_time = time.time()
    duration = end_time - start_time

    print("=" * 40)
    if success:
        print(f"[SUCCESS] 测试成功完成，耗时: {duration:.2f}秒")
    else:
        print(f"[FAILED] 测试失败，耗时: {duration:.2f}秒")

    return 0 if success else 1


if __name__ == '__main__':
    exit(main())