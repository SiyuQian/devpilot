#!/usr/bin/env python3
"""
汇率查询脚本 - 支持任意货币对之间的汇率查询和金额换算
默认以 NZD 为基准货币
"""

import requests
import sys
from datetime import datetime

# 货币名称映射
CURRENCY_NAMES = {
    "CNY": "人民币",
    "NZD": "纽币",
    "AUD": "澳币",
    "USD": "美元",
    "EUR": "欧元",
    "GBP": "英镑",
    "JPY": "日元",
    "KRW": "韩元",
    "HKD": "港币",
    "TWD": "台币",
    "SGD": "新加坡元",
    "CAD": "加元",
    "CHF": "瑞士法郎",
    "THB": "泰铢",
}


def _fetch_rates():
    """获取最新汇率数据（USD 为基准）"""
    url = "https://cdn.moneyconvert.net/api/latest.json"
    response = requests.get(url, timeout=10)
    response.raise_for_status()
    return response.json()


def _currency_label(code):
    """返回 '货币代码(中文名)' 格式"""
    name = CURRENCY_NAMES.get(code)
    return f"{code}({name})" if name else code


def get_exchange_rate(target_currency="CNY", base_currency="NZD", amount=1):
    """
    获取任意两种货币之间的汇率

    Args:
        target_currency: 目标货币代码 (默认 CNY)
        base_currency: 基准货币代码 (默认 NZD)
        amount: 换算金额 (默认 1)

    Returns:
        dict: 包含汇率信息的字典
    """
    try:
        data = _fetch_rates()

        # USD 是 API 的基准，汇率为 1
        rates = data["rates"]
        rates["USD"] = 1.0

        if base_currency not in rates:
            return {"success": False, "error": f"不支持的基准货币: {base_currency}"}
        if target_currency not in rates:
            return {"success": False, "error": f"不支持的目标货币: {target_currency}"}

        # 通过 USD 中转计算任意货币对
        rate = rates[target_currency] / rates[base_currency]
        converted = amount * rate

        query_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

        return {
            "success": True,
            "base_currency": base_currency,
            "target_currency": target_currency,
            "rate": round(rate, 4),
            "amount": amount,
            "converted": round(converted, 4),
            "timestamp": query_time,
            "source": "cdn.moneyconvert.net",
        }

    except requests.exceptions.RequestException as e:
        return {"success": False, "error": f"网络请求失败: {str(e)}"}
    except KeyError as e:
        return {"success": False, "error": f"货币代码无效: {str(e)}"}
    except Exception as e:
        return {"success": False, "error": f"未知错误: {str(e)}"}


def get_all_rates(base_currency="NZD", targets=None):
    """
    获取基准货币到多个目标货币的汇率

    Args:
        base_currency: 基准货币代码 (默认 NZD)
        targets: 目标货币列表 (默认 CNY, AUD, USD)

    Returns:
        dict: 包含多个汇率信息的字典
    """
    if targets is None:
        targets = ["CNY", "AUD", "USD"]
    # 排除基准货币自身
    targets = [t for t in targets if t != base_currency]

    try:
        data = _fetch_rates()
        rates_data = data["rates"]
        rates_data["USD"] = 1.0

        if base_currency not in rates_data:
            return {"success": False, "error": f"不支持的基准货币: {base_currency}"}

        base_rate = rates_data[base_currency]
        rates = {}
        for t in targets:
            if t in rates_data:
                rates[t] = round(rates_data[t] / base_rate, 4)

        query_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

        return {
            "success": True,
            "base_currency": base_currency,
            "rates": rates,
            "timestamp": query_time,
            "source": "cdn.moneyconvert.net",
        }

    except Exception as e:
        return {"success": False, "error": str(e)}


def get_cross_rates(currencies=None):
    """
    获取多个货币之间的交叉汇率表

    Args:
        currencies: 货币列表 (默认 NZD, AUD, USD, CNY)

    Returns:
        dict: 包含交叉汇率矩阵的字典
    """
    if currencies is None:
        currencies = ["NZD", "AUD", "USD", "CNY"]

    try:
        data = _fetch_rates()
        rates_data = data["rates"]
        rates_data["USD"] = 1.0

        matrix = {}
        for base in currencies:
            if base not in rates_data:
                continue
            matrix[base] = {}
            for target in currencies:
                if target not in rates_data:
                    continue
                if base == target:
                    matrix[base][target] = 1.0
                else:
                    matrix[base][target] = round(rates_data[target] / rates_data[base], 4)

        query_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

        return {
            "success": True,
            "currencies": currencies,
            "matrix": matrix,
            "timestamp": query_time,
            "source": "cdn.moneyconvert.net",
        }

    except Exception as e:
        return {"success": False, "error": str(e)}


def format_output(result):
    """格式化输出结果"""
    if not result["success"]:
        return f"查询失败: {result['error']}"

    if "matrix" in result:
        # 交叉汇率表输出
        currencies = result["currencies"]
        header = "         " + "  ".join(f"{c:>10}" for c in currencies)
        lines = ["汇率交叉表", header]
        for base in currencies:
            row = result["matrix"].get(base, {})
            vals = "  ".join(f"{row.get(t, '-'):>10}" for t in currencies)
            lines.append(f"{base:>8} {vals}")
        lines.extend([
            "",
            f"更新时间: {result['timestamp']}",
            f"数据来源: {result['source']}",
        ])
        return "\n".join(lines)

    if "rates" in result:
        # 多货币输出
        base = result["base_currency"]
        lines = [
            "汇率查询结果",
            f"基准货币: {_currency_label(base)}",
            "",
            f"汇率 (1 {base} = ):",
        ]
        for currency, rate in result["rates"].items():
            lines.append(f"  {_currency_label(currency)}: {rate}")
        lines.extend([
            "",
            f"更新时间: {result['timestamp']}",
            f"数据来源: {result['source']}",
        ])
        return "\n".join(lines)

    # 单货币输出（含金额换算）
    base = result["base_currency"]
    target = result["target_currency"]
    amount = result.get("amount", 1)
    converted = result.get("converted", result["rate"])

    lines = [
        "汇率查询结果",
        f"1 {_currency_label(base)} = {result['rate']} {_currency_label(target)}",
    ]
    if amount != 1:
        lines.append(f"{amount} {base} = {converted} {target}")
    lines.extend([
        "",
        f"更新时间: {result['timestamp']}",
        f"数据来源: {result['source']}",
    ])
    return "\n".join(lines)


if __name__ == "__main__":
    args = sys.argv[1:]

    if not args:
        # 无参数：显示 NZD 的常用汇率
        result = get_all_rates()
        print(format_output(result))
    elif args[0] == "--cross":
        # 交叉汇率表
        currencies = args[1:] if len(args) > 1 else None
        result = get_cross_rates(currencies)
        print(format_output(result))
    elif args[0] == "--base":
        # 指定基准货币查看常用汇率: --base AUD
        base = args[1].upper() if len(args) > 1 else "NZD"
        targets = [a.upper() for a in args[2:]] if len(args) > 2 else None
        result = get_all_rates(base_currency=base, targets=targets)
        print(format_output(result))
    else:
        # 简易用法: [金额] 基准货币 目标货币
        # 例: python3 get_rate.py AUD CNY
        #     python3 get_rate.py 100 AUD CNY
        amount = 1
        idx = 0
        try:
            amount = float(args[0])
            idx = 1
        except ValueError:
            pass

        base = args[idx].upper() if len(args) > idx else "NZD"
        target = args[idx + 1].upper() if len(args) > idx + 1 else "CNY"
        result = get_exchange_rate(target_currency=target, base_currency=base, amount=amount)
        print(format_output(result))
