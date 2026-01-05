package main

// Common X11 keysyms (from X11/keysymdef.h)
const (
	XK_BackSpace = 0xff08
	XK_Tab       = 0xff09
	XK_Return    = 0xff0d
	XK_Escape    = 0xff1b
	XK_Delete    = 0xffff

	XK_Home  = 0xff50
	XK_Left  = 0xff51
	XK_Up    = 0xff52
	XK_Right = 0xff53
	XK_Down  = 0xff54
	XK_End   = 0xff57

	XK_Print = 0xff61

	XK_F1  = 0xffbe
	XK_F2  = 0xffbf
	XK_F3  = 0xffc0
	XK_F4  = 0xffc1
	XK_F5  = 0xffc2
	XK_F6  = 0xffc3
	XK_F7  = 0xffc4
	XK_F8  = 0xffc5
	XK_F9  = 0xffc6
	XK_F10 = 0xffc7
	XK_F11 = 0xffc8
	XK_F12 = 0xffc9

	XK_Shift_L   = 0xffe1
	XK_Shift_R   = 0xffe2
	XK_Control_L = 0xffe3
	XK_Control_R = 0xffe4
	XK_Alt_L     = 0xffe9
	XK_Alt_R     = 0xffea
	XK_Super_L   = 0xffeb
	XK_Super_R   = 0xffec

	XK_space      = 0x0020
	XK_exclam     = 0x0021
	XK_quotedbl   = 0x0022
	XK_numbersign = 0x0023
	XK_dollar     = 0x0024
	XK_percent    = 0x0025
	XK_ampersand  = 0x0026
	XK_apostrophe = 0x0027
	XK_parenleft  = 0x0028
	XK_parenright = 0x0029
	XK_asterisk   = 0x002a
	XK_plus       = 0x002b
	XK_comma      = 0x002c
	XK_minus      = 0x002d
	XK_period     = 0x002e
	XK_slash      = 0x002f

	XK_0 = 0x0030
	XK_1 = 0x0031
	XK_2 = 0x0032
	XK_3 = 0x0033
	XK_4 = 0x0034
	XK_5 = 0x0035
	XK_6 = 0x0036
	XK_7 = 0x0037
	XK_8 = 0x0038
	XK_9 = 0x0039

	XK_colon     = 0x003a
	XK_semicolon = 0x003b
	XK_less      = 0x003c
	XK_equal     = 0x003d
	XK_greater   = 0x003e
	XK_question  = 0x003f
	XK_at        = 0x0040

	XK_A = 0x0041
	XK_B = 0x0042
	XK_C = 0x0043
	XK_D = 0x0044
	XK_E = 0x0045
	XK_F = 0x0046
	XK_G = 0x0047
	XK_H = 0x0048
	XK_I = 0x0049
	XK_J = 0x004a
	XK_K = 0x004b
	XK_L = 0x004c
	XK_M = 0x004d
	XK_N = 0x004e
	XK_O = 0x004f
	XK_P = 0x0050
	XK_Q = 0x0051
	XK_R = 0x0052
	XK_S = 0x0053
	XK_T = 0x0054
	XK_U = 0x0055
	XK_V = 0x0056
	XK_W = 0x0057
	XK_X = 0x0058
	XK_Y = 0x0059
	XK_Z = 0x005a

	XK_bracketleft  = 0x005b
	XK_backslash    = 0x005c
	XK_bracketright = 0x005d
	XK_asciicircum  = 0x005e
	XK_underscore   = 0x005f
	XK_grave        = 0x0060

	XK_a = 0x0061
	XK_b = 0x0062
	XK_c = 0x0063
	XK_d = 0x0064
	XK_e = 0x0065
	XK_f = 0x0066
	XK_g = 0x0067
	XK_h = 0x0068
	XK_i = 0x0069
	XK_j = 0x006a
	XK_k = 0x006b
	XK_l = 0x006c
	XK_m = 0x006d
	XK_n = 0x006e
	XK_o = 0x006f
	XK_p = 0x0070
	XK_q = 0x0071
	XK_r = 0x0072
	XK_s = 0x0073
	XK_t = 0x0074
	XK_u = 0x0075
	XK_v = 0x0076
	XK_w = 0x0077
	XK_x = 0x0078
	XK_y = 0x0079
	XK_z = 0x007a

	XK_braceleft  = 0x007b
	XK_bar        = 0x007c
	XK_braceright = 0x007d
	XK_asciitilde = 0x007e

	// XF86 multimedia keys
	XF86XK_AudioMute        = 0x1008ff12
	XF86XK_AudioLowerVolume = 0x1008ff11
	XF86XK_AudioRaiseVolume = 0x1008ff13
	XF86XK_AudioPlay        = 0x1008ff14
	XF86XK_AudioStop        = 0x1008ff15
	XF86XK_AudioPrev        = 0x1008ff16
	XF86XK_AudioNext        = 0x1008ff17
	XF86XK_MonBrightnessUp   = 0x1008ff02
	XF86XK_MonBrightnessDown = 0x1008ff03
)
