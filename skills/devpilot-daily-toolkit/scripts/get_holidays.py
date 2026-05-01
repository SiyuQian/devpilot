#!/usr/bin/env python3
"""
Query New Zealand public holidays using the free Nager.Date API.
Usage: python3 get_holidays.py [year]
"""

import sys
import json
import urllib.request
from datetime import datetime

def get_nz_holidays(year=None):
    """Get New Zealand public holidays for a given year."""
    if year is None:
        year = datetime.now().year
    
    url = f"https://date.nager.at/api/v3/publicholidays/{year}/NZ"
    
    try:
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=10) as response:
            data = json.loads(response.read().decode('utf-8'))
            return data
    except Exception as e:
        print(f"Error fetching holidays: {e}", file=sys.stderr)
        sys.exit(1)

# 奥克兰地区代码
AUCKLAND_REGIONS = ["NZ-AUK", "NZ-NTL", "NZ-WKO", "NZ-GIS", "NZ-BOP"]

def is_auckland_holiday(holiday):
    """Check if a holiday applies to Auckland region."""
    counties = holiday.get('counties', []) or []
    return any(c in AUCKLAND_REGIONS for c in counties)

def format_holidays(holidays, show_all=False, highlight_auckland=True):
    """Format holidays for display."""
    today = datetime.now().date()
    
    # Separate global and regional holidays
    global_holidays = [h for h in holidays if h.get('global')]
    regional_holidays = [h for h in holidays if not h.get('global')]
    
    # Filter Auckland holidays if highlighting
    auckland_holidays = [h for h in regional_holidays if is_auckland_holiday(h)]
    other_regional = [h for h in regional_holidays if not is_auckland_holiday(h)]
    
    result = []
    result.append("🇳🇿 新西兰公共假期")
    result.append("=" * 40)
    result.append("")
    
    # National holidays
    result.append("📅 全国性假期")
    result.append("-" * 40)
    
    for holiday in global_holidays:
        date_str = holiday['date']
        name = holiday['localName']
        
        # Parse date
        date_obj = datetime.strptime(date_str, "%Y-%m-%d").date()
        days_until = (date_obj - today).days
        
        # Format date nicely
        date_formatted = date_obj.strftime("%m月%d日")
        
        # Add indicator for past/upcoming
        if days_until < 0:
            indicator = "✅"
        elif days_until == 0:
            indicator = "🎉 今天!"
        elif days_until <= 7:
            indicator = f"⏳ {days_until}天后"
        else:
            indicator = ""
        
        line = f"  {date_formatted} - {name}"
        if indicator:
            line += f" {indicator}"
        result.append(line)
    
    # Auckland regional holidays
    if highlight_auckland and auckland_holidays:
        result.append("")
        result.append("🏠 奥克兰地区假期")
        result.append("-" * 40)
        
        for holiday in auckland_holidays:
            date_str = holiday['date']
            name = holiday['localName']
            date_obj = datetime.strptime(date_str, "%Y-%m-%d").date()
            days_until = (date_obj - today).days
            date_formatted = date_obj.strftime("%m月%d日")
            
            # Add indicator for past/upcoming
            if days_until < 0:
                indicator = "✅"
            elif days_until == 0:
                indicator = "🎉 今天!"
            elif days_until <= 7:
                indicator = f"⏳ {days_until}天后"
            else:
                indicator = ""
            
            line = f"  {date_formatted} - {name}"
            if indicator:
                line += f" {indicator}"
            result.append(line)
    
    if show_all and other_regional:
        result.append("")
        result.append("📍 其他地区假期")
        result.append("-" * 40)
        
        for holiday in other_regional:
            date_str = holiday['date']
            name = holiday['localName']
            date_obj = datetime.strptime(date_str, "%Y-%m-%d").date()
            date_formatted = date_obj.strftime("%m月%d日")
            result.append(f"  {date_formatted} - {name}")
    
    return "\n".join(result)

def get_next_holiday(holidays, include_auckland_only=True):
    """Get the next upcoming holiday."""
    today = datetime.now().date()
    
    # Filter holidays relevant to Auckland user
    relevant_holidays = []
    for holiday in holidays:
        date_obj = datetime.strptime(holiday['date'], "%Y-%m-%d").date()
        if date_obj >= today:
            # Include if it's a national holiday or Auckland regional holiday
            if holiday.get('global') or (include_auckland_only and is_auckland_holiday(holiday)):
                relevant_holidays.append(holiday)
    
    # Return the closest one
    if relevant_holidays:
        return min(relevant_holidays, key=lambda h: datetime.strptime(h['date'], "%Y-%m-%d").date())
    return None

if __name__ == "__main__":
    year = None
    show_all = False
    
    if len(sys.argv) > 1:
        if sys.argv[1] in ['--all', '-a']:
            show_all = True
        else:
            try:
                year = int(sys.argv[1])
            except ValueError:
                print(f"Usage: {sys.argv[0]} [year|--all]", file=sys.stderr)
                sys.exit(1)
    
    if len(sys.argv) > 2 and sys.argv[2] in ['--all', '-a']:
        show_all = True
    
    holidays = get_nz_holidays(year)
    
    # Print formatted list (highlight Auckland by default)
    print(format_holidays(holidays, show_all, highlight_auckland=True))
    
    # Show next upcoming holiday (relevant to Auckland)
    next_holiday = get_next_holiday(holidays, include_auckland_only=True)
    if next_holiday:
        print("")
        print("🎯 下一个假期")
        print("-" * 40)
        date_obj = datetime.strptime(next_holiday['date'], "%Y-%m-%d").date()
        days_until = (date_obj - datetime.now().date()).days
        holiday_name = next_holiday['localName']
        
        # Add location indicator
        if next_holiday.get('global'):
            location = "全国性假期"
        elif is_auckland_holiday(next_holiday):
            location = "奥克兰地区假期"
        else:
            location = "地区性假期"
        
        print(f"  {holiday_name}")
        print(f"  日期: {date_obj.strftime('%Y年%m月%d日')}")
        print(f"  类型: {location}")
        if days_until == 0:
            print("  🎉 就是今天!")
        else:
            print(f"  ⏳ 还有 {days_until} 天")
