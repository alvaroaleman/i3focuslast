# i3 focus last


```
exec i3focuslast
bindsym $alt+Tab exec curl --unix-socket /tmp/.focus_last http://love
```
