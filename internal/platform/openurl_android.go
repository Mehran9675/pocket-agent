//go:build android

package platform

import (
	"gioui.org/app"
	"git.wow.st/gmp/jni"
)

// openURL starts an ACTION_VIEW intent from the application context.
func openURL(url string) error {
	vm := jni.JVMFor(app.JavaVM())
	return jni.Do(vm, func(env jni.Env) error {
		ctx := jni.Object(app.AppContext())
		uriCls := jni.FindClass(env, "android/net/Uri")
		parse := jni.GetStaticMethodID(env, uriCls, "parse", "(Ljava/lang/String;)Landroid/net/Uri;")
		uri, err := jni.CallStaticObjectMethod(env, uriCls, parse, jni.Value(jni.JavaString(env, url)))
		if err != nil {
			return err
		}
		intentCls := jni.FindClass(env, "android/content/Intent")
		ctor := jni.GetMethodID(env, intentCls, "<init>", "(Ljava/lang/String;Landroid/net/Uri;)V")
		intent, err := jni.NewObject(env, intentCls, ctor, jni.Value(jni.JavaString(env, "android.intent.action.VIEW")), jni.Value(uri))
		if err != nil {
			return err
		}
		addFlags := jni.GetMethodID(env, intentCls, "addFlags", "(I)Landroid/content/Intent;")
		const flagActivityNewTask = 0x10000000
		if _, err := jni.CallObjectMethod(env, intent, addFlags, jni.Value(flagActivityNewTask)); err != nil {
			return err
		}
		ctxCls := jni.GetObjectClass(env, ctx)
		start := jni.GetMethodID(env, ctxCls, "startActivity", "(Landroid/content/Intent;)V")
		return jni.CallVoidMethod(env, ctx, start, jni.Value(intent))
	})
}
