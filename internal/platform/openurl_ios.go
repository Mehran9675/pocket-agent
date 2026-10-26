//go:build ios

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UIKit -framework Foundation
#include <stdlib.h>
#import <UIKit/UIKit.h>

static void pa_open_url(const char *s) {
	NSURL *url = [NSURL URLWithString:[NSString stringWithUTF8String:s]];
	if (url == nil) return;
	dispatch_async(dispatch_get_main_queue(), ^{
		[[UIApplication sharedApplication] openURL:url options:@{} completionHandler:nil];
	});
}
*/
import "C"

import "unsafe"

func openURL(url string) error {
	cs := C.CString(url)
	defer C.free(unsafe.Pointer(cs))
	C.pa_open_url(cs)
	return nil
}
