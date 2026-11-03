window.XBOARD_TRANSLATIONS = window.XBOARD_TRANSLATIONS || {};
window.XBOARD_TRANSLATIONS['vi-VN'] = {
  "theme": {
    "title": "Cấu hình giao diện",
    "description": "Cấu hình giao diện, bao gồm màu sắc, kích thước phông chữ, v.v. Nếu bạn triển khai Lets VPN với frontend và backend tách biệt, cấu hình giao diện sẽ không có hiệu lực.",
    "upload": {
      "button": "Tải lên giao diện",
      "title": "Tải lên giao diện",
      "description": "Vui lòng tải lên gói giao diện hợp lệ (định dạng .zip). Gói phải chứa đầy đủ cấu trúc tệp của giao diện.",
      "dragText": "Kéo tệp giao diện vào đây hoặc",
      "clickText": "nhấn để chọn",
      "supportText": "Hỗ trợ các gói giao diện định dạng .zip",
      "uploading": "Đang tải lên...",
      "error": {
        "format": "Chỉ hỗ trợ tệp giao diện định dạng ZIP"
      }
    },
    "preview": {
      "title": "Xem trước giao diện",
      "imageCount": "{{current}} / {{total}}"
    },
    "card": {
      "version": "Phiên bản: {{version}}",
      "currentTheme": "Giao diện hiện tại",
      "activateTheme": "Kích hoạt",
      "configureTheme": "Thiết lập",
      "preview": "Xem trước",
      "delete": {
        "title": "Xóa giao diện",
        "description": "Bạn có chắc chắn muốn xóa giao diện này? Hành động này không thể hoàn tác.",
        "button": "Xóa",
        "error": {
          "active": "Không thể xóa giao diện đang hoạt động"
        }
      }
    },
    "config": {
      "title": "Cài đặt giao diện {{name}}",
      "description": "Thay đổi kiểu dáng, bố cục và các tham số hiển thị khác của giao diện.",
      "cancel": "Hủy",
      "save": "Lưu",
      "success": "Đã lưu cài đặt thành công"
    }
  },
  "machine": {
    "title": "Quản lý máy chủ",
    "description": "Xem trạng thái máy chủ, tải trọng và các node được lưu trữ, cũng như thực hiện các thao tác với node từ chế độ xem máy chủ.",
    "columns": {
      "id": "ID",
      "name": "Tên máy chủ",
      "status": "Trạng thái",
      "nodes": "Node",
      "nodesHosted": "node đang chạy",
      "nodesIdle": "không có node",
      "load": "Tải trọng",
      "lastSeen": "Phản hồi cuối",
      "actions": "Thao tác",
      "online": "Trực tuyến",
      "offline": "Ngoại tuyến",
      "inactive": "Vô hiệu hóa",
      "noData": "Không có dữ liệu tải trọng",
      "cpu": "CPU",
      "memory": "RAM",
      "disk": "Disk",
      "never": "Không bao giờ",
      "lastReport": "Báo cáo tải trọng"
    },
    "toolbar": {
      "search": "Tìm theo tên máy chủ, ghi chú hoặc SID...",
      "status": "Trạng thái",
      "nodes": "Node",
      "nodesHosted": "node đang chạy",
      "nodesIdle": "node trống",
      "with_nodes": "Có node đang chạy",
      "idle_nodes": "Máy chủ trống",
      "high_load": "Tải cao",
      "online_ratio": "Trực tuyến",
      "high_load_count": "Số lượng tải cao",
      "tip": "Trang này giúp đánh giá nhanh trạng thái máy chủ, số lượng node và tải tài nguyên.",
      "reset": "Đặt lại"
    },
    "overview": {
      "total": "Tổng máy chủ",
      "total_hint": "Tổng số node đang chạy: {{count}}",
      "online": "Trực tuyến",
      "online_hint": "Có tín hiệu heartbeat trong 5 phút qua",
      "offline": "Ngoại tuyến",
      "offline_hint": "Cần kiểm tra heartbeat hoặc agent",
      "high_load": "Tải cao",
      "high_load_hint": "CPU, RAM hoặc Disk gần đạt ngưỡng giới hạn",
      "nodes_suffix": "node",
      "attention": "Cảnh báo",
      "stable": "Ổn định",
      "needs_review": "Cần kiểm tra",
      "normal": "Bình thường"
    },
    "form": {
      "add": "Thêm máy chủ",
      "create": "Máy chủ mới",
      "edit": "Sửa máy chủ",
      "createDescription": "Tạo máy chủ mới để quản lý nhiều node.",
      "editDescription": "Thay đổi tên, ghi chú hoặc trạng thái máy chủ.",
      "name": "Tên máy chủ",
      "namePlaceholder": "Ví dụ: HK-01",
      "nameError": "Vui lòng nhập tên máy chủ",
      "notes": "Ghi chú",
      "notesPlaceholder": "Ghi chú tùy chọn về máy chủ",
      "isActive": "Bật máy chủ",
      "isActiveDescription": "Các máy chủ bị tắt sẽ không được xboard-node sử dụng.",
      "cancel": "Hủy",
      "submit": "Lưu",
      "update": "Cập nhật"
    },
    "token": {
      "title": "Token máy chủ",
      "description": "Token này được xboard-node sử dụng để xác thực. Vui lòng giữ bí mật.",
      "show": "Hiện token",
      "hide": "Ẩn token",
      "reset": "Đặt lại token",
      "resetConfirm": "Đặt lại token?",
      "resetDescription": "Token cũ sẽ bị vô hiệu hóa ngay lập tức. xboard-node sẽ cần cấu hình lại.",
      "copy": "Sao chép",
      "copied": "Đã sao chép token",
      "copiedInline": "Đã sao chép!",
      "copyFailed": "Lỗi sao chép",
      "autoHide": "Sẽ ẩn sau {{time}}",
      "resetSuccess": "Đã đặt lại token",
      "createdHint": "Token đã được tạo. Bạn có thể xem lại bất cứ lúc nào trên trang máy chủ."
    },
    "install": {
      "title": "Cài đặt xboard-node",
      "description": "Chạy lệnh này trên máy chủ mục tiêu để cài đặt xboard-node ở chế độ machine mode và kết nối nó với bản ghi máy chủ hiện tại.",
      "copy": "Sao chép lệnh cài đặt",
      "copied": "Đã sao chép lệnh cài đặt",
      "copiedInline": "Đã sao chép!",
      "copyFailed": "Lỗi sao chép",
      "loading": "Đang tạo lệnh...",
      "hint": "Yêu cầu quyền root hoặc sudo, và máy chủ Linux chạy systemd."
    },
    "detail": {
      "title": "Chi tiết máy chủ",
      "info": "Thông tin",
      "associatedNodes": "Node liên kết",
      "noNodes": "Không có node nào được liên kết với máy chủ này.",
      "nodeId": "ID",
      "nodeName": "Tên",
      "nodeType": "Loại",
      "nodeHost": "Địa chỉ",
      "nodePort": "Cổng",
      "nodeShow": "Hiển thị",
      "nodeEnabled": "Kích hoạt",
      "loadTrend": "Xu hướng tải",
      "networkTrend": "Tốc độ mạng",
      "noHistory": "Chưa có lịch sử tải trọng.",
      "openNodeManage": "Mở quản lý node",
      "addNodeToServer": "Thêm node vào máy chủ này",
      "nodeCount": "{{count}} node",
      "nodeEnabledCount": "{{count}} đang hoạt động",
      "toggleEnabledError": "Không thể chuyển trạng thái node",
      "bindExistingButton": "Liên kết node có sẵn",
      "bindExistingTitle": "Liên kết node có sẵn",
      "bindExistingDescription": "Chọn các node để liên kết với «{{name}}»",
      "bindSearchPlaceholder": "Tìm theo tên, địa chỉ, loại...",
      "noUnboundNodes": "Không có node trống",
      "noSearchResults": "Không tìm thấy kết quả",
      "selectAll": "Chọn tất cả ({{count}})",
      "selectedCount": "Đã chọn {{count}}",
      "bindConfirm": "Liên kết {{count}} node",
      "binding": "Đang liên kết...",
      "bindSuccess": "Đã liên kết thành công {{count}} node với «{{name}}»",
      "bindFailed": "Lỗi liên kết",
      "unbindNode": "Hủy liên kết",
      "unbindSuccess": "«{{name}}» đã hủy liên kết",
      "unbindFailed": "Lỗi hủy liên kết",
      "cancel": "Hủy"
    },
    "messages": {
      "createSuccess": "Tạo máy chủ thành công",
      "updateSuccess": "Cập nhật máy chủ thành công",
      "deleteConfirm": "Xóa máy chủ?",
      "deleteDescription": "Các node liên kết sẽ bị hủy liên kết (không bị xóa). Hành động này không thể đảo ngược.",
      "deleteButton": "Xóa",
      "deleteSuccess": "Xóa máy chủ thành công",
      "deleteFailed": "Không thể xóa máy chủ",
      "saveFailed": "Không thể lưu máy chủ",
      "tokenFetchFailed": "Không thể lấy token",
      "tokenResetFailed": "Không thể đặt lại token"
    },
    "nodeForm": {
      "machineId": "Liên kết với máy chủ",
      "machineIdPlaceholder": "Chọn máy chủ (tùy chọn)",
      "machineIdNone": "Triển khai độc lập",
      "enabled": "Kích hoạt trên máy chủ",
      "enabledDescription": "Chạy node trên máy chủ đã liên kết"
    }
  },
  "common": {
    "loading": "Đang tải...",
    "error": "Lỗi",
    "success": "Thành công",
    "save": "Lưu",
    "cancel": "Hủy",
    "confirm": "Xác nhận",
    "close": "Đóng",
    "delete": {
      "success": "Xóa thành công",
      "failed": "Không thể xóa"
    },
    "edit": "Sửa",
    "view": "Xem",
    "toggleNavigation": "Chuyển đổi điều hướng",
    "toggleSidebar": "Chuyển đổi thanh bên",
    "search": "Tìm kiếm...",
    "theme": {
      "light": "Sáng",
      "dark": "Tối",
      "system": "Hệ thống"
    },
    "user": "Người dùng",
    "defaultEmail": "user@example.com",
    "settings": "Cài đặt",
    "logout": "Đăng xuất",
    "copy": {
      "success": "Sao chép thành công",
      "failed": "Không thể sao chép",
      "error": "Lỗi sao chép",
      "errorLog": "Lỗi khi sao chép vào clipboard"
    },
    "submit": "Gửi",
    "saving": "Đang lưu...",
    "table": {
      "noData": "Không có dữ liệu",
      "pagination": {
        "selected": "Đã chọn {{selected}} trong tổng số {{total}}",
        "itemsPerPage": "Số item mỗi trang",
        "page": "Trang",
        "pageOf": "trên {{total}}",
        "firstPage": "Về trang đầu",
        "previousPage": "Trang trước",
        "nextPage": "Trang sau",
        "lastPage": "Về trang cuối"
      },
      "viewOptions": {
        "button": "Cột",
        "label": "Chuyển đổi hiển thị cột"
      }
    },
    "update": {
      "title": "Cập nhật hệ thống",
      "newVersion": "Có phiên bản mới",
      "currentVersion": "Phiên bản hiện tại",
      "latestVersion": "Phiên bản mới nhất",
      "updateLater": "Cập nhật sau",
      "updateNow": "Cập nhật ngay",
      "updating": "Đang cập nhật...",
      "updateSuccess": "Cập nhật thành công, hệ thống sẽ khởi động lại shortly",
      "updateFailed": "Cập nhật thất bại, vui lòng thử lại sau"
    },
    "time": {
      "day": "ngày",
      "hour": " giờ"
    },
    "reset": "Đặt lại",
    "export": "Xuất",
    "currency": {
      "yuan": "Nhân dân tệ"
    },
    "http": {
      "notLoggedIn": "Chưa đăng nhập",
      "unknownError": "Lỗi không xác định",
      "loginExpired": "Phiên đăng nhập hết hạn",
      "noPermission": "Không có quyền",
      "notFound": "Không tìm thấy tài nguyên hoặc API",
      "unknownException": "Ngoại lệ không xác định"
    }
  },
  "plugin": {
    "title": "Quản lý Plugin",
    "description": "Quản lý và cấu hình các plugin hệ thống",
    "search": {
      "placeholder": "Tìm theo tên hoặc mô tả..."
    },
    "type": {
      "placeholder": "Chọn loại plugin",
      "all": "Tất cả loại"
    },
    "tabs": {
      "all": "Tất cả plugin",
      "installed": "Đã cài đặt",
      "available": "Khả dụng"
    },
    "status": {
      "enabled": "Đã bật",
      "disabled": "Đã tắt",
      "not_installed": "Chưa cài đặt",
      "protected": "Được bảo vệ",
      "filter_placeholder": "Lọc trạng thái",
      "all": "Tất cả trạng thái",
      "installed": "Đã cài đặt",
      "available": "Khả dụng"
    },
    "button": {
      "install": "Cài đặt",
      "upgrade": "Nâng cấp",
      "config": "Cấu hình",
      "enable": "Bật",
      "disable": "Tắt",
      "uninstall": "Gỡ cài đặt",
      "readme": "Tài liệu",
      "menuDemo": "Demo Menu"
    },
    "upload": {
      "button": "Tải lên Plugin",
      "title": "Tải lên Plugin",
      "description": "Tải lên gói plugin (.zip)",
      "dragText": "Kéo gói plugin vào đây hoặc",
      "clickText": "chọn tệp",
      "supportText": "Chỉ hỗ trợ tệp .zip",
      "uploading": "Đang tải lên...",
      "error": {
        "format": "Chỉ hỗ trợ tệp .zip"
      }
    },
    "delete": {
      "title": "Xóa Plugin",
      "description": "Bạn có chắc chắn muốn xóa plugin này? Hành động này không thể hoàn tác.",
      "button": "Xóa"
    },
    "uninstall": {
      "title": "Gỡ cài đặt Plugin khỏi hệ thống",
      "description": "Bạn có chắc chắn muốn gỡ cài đặt plugin này? Tất cả dữ liệu plugin sẽ bị xóa.",
      "button": "Gỡ cài đặt"
    },
    "upgrade": {
      "title": "Nâng cấp Plugin",
      "description": "Bạn có chắc chắn muốn nâng cấp plugin này? Nó sẽ tạm thời không khả dụng trong quá trình này.",
      "button": "Nâng cấp"
    },
    "config": {
      "title": "Cấu hình",
      "description": "Thay đổi cài đặt plugin",
      "save": "Lưu",
      "cancel": "Hủy"
    },
    "readme": {
      "title": "Tài liệu Plugin"
    },
    "author": "Tác giả",
    "messages": {
      "installSuccess": "Cài đặt plugin thành công",
      "installError": "Lỗi khi cài đặt plugin",
      "upgradeSuccess": "Nâng cấp plugin thành công",
      "upgradeError": "Lỗi khi nâng cấp plugin",
      "uninstallSuccess": "Gỡ cài đặt plugin thành công",
      "uninstallError": "Lỗi khi gỡ cài đặt plugin",
      "enableSuccess": "Đã bật plugin",
      "enableError": "Lỗi khi bật plugin",
      "disableSuccess": "Đã tắt plugin",
      "disableError": "Lỗi khi tắt plugin",
      "configLoadError": "Lỗi khi tải cấu hình plugin",
      "configSaveSuccess": "Đã lưu cấu hình",
      "configSaveError": "Lỗi khi lưu cấu hình",
      "uploadSuccess": "Đã tải lên plugin",
      "uploadError": "Lỗi khi tải lên plugin",
      "deleteSuccess": "Đã xóa plugin",
      "deleteError": "Lỗi khi xóa plugin"
    }
  },
  "group": {
    "title": "Nhóm quyền",
    "description": "Quản lý tất cả các nhóm quyền, bao gồm thêm, xóa và sửa.",
    "columns": {
      "id": "ID Nhóm",
      "name": "Tên nhóm",
      "usersCount": "Người dùng",
      "serverCount": "Node",
      "actions": "Thao tác"
    },
    "form": {
      "add": "Thêm nhóm",
      "edit": "Sửa nhóm",
      "create": "Tạo nhóm",
      "update": "Cập nhật",
      "name": "Tên nhóm",
      "namePlaceholder": "Nhập tên nhóm",
      "nameDescription": "Tên nhóm dùng để nhận diện. Khuyến nghị sử dụng tên dễ hiểu.",
      "cancel": "Hủy",
      "editDescription": "Thay đổi thông tin nhóm. Thay đổi có hiệu lực ngay lập tức.",
      "createDescription": "Tạo nhóm quyền mới để phân quyền truy cập người dùng."
    },
    "toolbar": {
      "searchPlaceholder": "Tìm nhóm...",
      "reset": "Đặt lại"
    },
    "messages": {
      "deleteConfirm": "Xác nhận xóa",
      "deleteDescription": "Hành động này sẽ xóa vĩnh viễn nhóm quyền này. Tiếp tục?",
      "deleteButton": "Xóa",
      "createSuccess": "Tạo thành công",
      "updateSuccess": "Cập nhật thành công",
      "nameValidation": {
        "min": "Tên phải có ít nhất 2 ký tự",
        "max": "Tên không được vượt quá 50 ký tự",
        "pattern": "Tên chỉ được chứa chữ cái, số, ký tự tiếng Trung, gạch dưới và gạch ngang"
      }
    }
  },
  "nav": {
    "dashboard": "Bảng điều khiển",
    "systemManagement": "Quản lý hệ thống",
    "systemConfig": "Cài đặt hệ thống",
    "themeConfig": "Cài đặt giao diện",
    "noticeManagement": "Quản lý thông báo",
    "pluginManagement": "Quản lý Plugin",
    "paymentConfig": "Cài đặt thanh toán",
    "knowledgeManagement": "Kho kiến thức",
    "nodeManagement": "Quản lý Node",
    "machineManagement": "Quản lý Máy chủ",
    "permissionGroupManagement": "Nhóm truy cập",
    "routeManagement": "Quản lý Route",
    "subscriptionManagement": "Quản lý Đăng ký",
    "planManagement": "Gói đăng ký",
    "orderManagement": "Quản lý Đơn hàng",
    "couponManagement": "Mã giảm giá",
    "giftCardManagement": "Thẻ quà tặng",
    "userManagement": "Người dùng",
    "trafficResetLogs": "Nhật ký Reset Traffic",
    "ticketManagement": "Ticket hỗ trợ",
    "pluginApps": "Ứng dụng Plugin"
  },
  "auth": {
    "signIn": {
      "heroTitle": "Kết nối an toàn, ổn định và nhanh",
      "heroSubtitle": "Quản lý node, người dùng, đơn hàng và gói đăng ký tại một nơi.",
      "heroPoint1": "Điều phối nhiều node với trạng thái trực tiếp",
      "heroPoint2": "Gói, đơn hàng và thanh toán tập trung",
      "heroPoint3": "Tự động nhắc lưu lượng và hết hạn",
      "title": "Đăng nhập",
      "description": "Nhập email và mật khẩu để đăng nhập",
      "email": "Email",
      "emailPlaceholder": "name@example.com",
      "password": "Mật khẩu",
      "passwordPlaceholder": "Nhập mật khẩu của bạn",
      "forgotPassword": "Quên mật khẩu?",
      "submit": "Đăng nhập",
      "rememberMe": "Ghi nhớ tôi",
      "resetPassword": {
        "title": "Đặt lại mật khẩu",
        "description": "Chạy lệnh sau trong thư mục gốc của trang web để đặt lại mật khẩu",
        "command": "php artisan reset:password admin-email"
      },
      "validation": {
        "emailRequired": "Vui lòng nhập địa chỉ email",
        "emailInvalid": "Vui lòng nhập địa chỉ email hợp lệ",
        "passwordRequired": "Vui lòng nhập mật khẩu",
        "passwordLength": "Mật khẩu phải có ít nhất 7 ký tự"
      }
    }
  },
  "notice": {
    "title": "Quản lý Thông báo",
    "description": "Tại đây bạn có thể cấu hình thông báo, bao gồm thêm, xóa, sửa và các thao tác khác.",
    "table": {
      "columns": {
        "id": "ID",
        "show": "Trạng thái hiển thị",
        "title": "Tiêu đề",
        "actions": "Thao tác"
      },
      "toolbar": {
        "search": "Tìm theo tiêu đề...",
        "reset": "Đặt lại",
        "sort": {
          "edit": "Sửa thứ tự",
          "save": "Lưu thứ tự"
        }
      },
      "actions": {
        "edit": "Sửa",
        "delete": {
          "title": "Xác nhận xóa",
          "description": "Bạn có chắc chắn muốn xóa thông báo này? Hành động này không thể hoàn tác.",
          "success": "Xóa thành công"
        }
      }
    },
    "form": {
      "add": {
        "title": "Thêm thông báo",
        "button": "Thêm thông báo"
      },
      "edit": {
        "title": "Sửa thông báo"
      },
      "fields": {
        "title": {
          "label": "Tiêu đề",
          "placeholder": "Nhập tiêu đề thông báo"
        },
        "content": {
          "label": "Nội dung"
        },
        "img_url": {
          "label": "Ảnh nền",
          "placeholder": "Nhập URL ảnh nền"
        },
        "show": {
          "label": "Hiển thị"
        },
        "tags": {
          "label": "Thẻ",
          "placeholder": "Nhấn Enter để thêm thẻ"
        }
      },
      "buttons": {
        "cancel": "Hủy",
        "submit": "Gửi",
        "success": "Gửi thành công"
      }
    }
  },
  "order": {
    "userFilter": {
      "active": "Đang xem đơn hàng của người dùng #{{id}}",
      "total": "Tổng {{count}}",
      "clear": "Hiện tất cả đơn hàng"
    },
    "title": "Quản lý Đơn hàng",
    "description": "Tại đây bạn có thể xem đơn hàng của người dùng, bao gồm gán, xem, xóa và các thao tác khác.",
    "table": {
      "columns": {
        "tradeNo": "Mã Đơn hàng",
        "type": "Loại",
        "plan": "Gói đăng ký",
        "period": "Chu kỳ",
        "amount": "Số tiền thanh toán",
        "status": "Trạng thái đơn hàng",
        "commission": "Hoa hồng",
        "commissionStatus": "Trạng thái hoa hồng",
        "createdAt": "Ngày tạo"
      }
    },
    "type": {
      "NEW": "Mua mới",
      "RENEWAL": "Gia hạn",
      "UPGRADE": "Nâng cấp",
      "RESET_FLOW": "Reset Traffic"
    },
    "period": {
      "month_price": "1 Tháng",
      "quarter_price": "3 Tháng",
      "half_year_price": "6 Tháng",
      "year_price": "1 năm",
      "two_year_price": "2 năm",
      "three_year_price": "3 năm",
      "onetime_price": "Vĩnh viễn",
      "reset_price": "Đặt lại dung lượng"
    },
    "status": {
      "PENDING": "Chờ xử lý",
      "PROCESSING": "Đang xử lý",
      "CANCELLED": "Đã hủy",
      "COMPLETED": "Hoàn thành",
      "DISCOUNTED": "Giảm giá",
      "tooltip": "Sau khi đánh dấu [Đã thanh toán], hệ thống sẽ kích hoạt và hoàn tất đơn hàng"
    },
    "commission": {
      "PENDING": "Chờ xử lý",
      "PROCESSING": "Đang xử lý",
      "VALID": "Hợp lệ",
      "INVALID": "Không hợp lệ"
    },
    "actions": {
      "markAsPaid": "Đánh dấu đã thanh toán",
      "cancel": "Hủy đơn hàng",
      "openMenu": "Mở menu",
      "reset": "Đặt lại"
    },
    "search": {
      "placeholder": "Tìm đơn hàng..."
    },
    "dialog": {
      "title": "Thông tin đơn hàng",
      "basicInfo": "Thông tin cơ bản",
      "amountInfo": "Thông tin chi phí",
      "timeInfo": "Mốc thời gian",
      "commissionInfo": "Thông tin hoa hồng",
      "commissionStatusActive": "Đang hoạt động",
      "addOrder": "Thêm đơn hàng",
      "assignOrder": "Gán đơn hàng",
      "fields": {
        "userEmail": "Email người dùng",
        "orderPeriod": "Chu kỳ đơn hàng",
        "subscriptionPlan": "Gói đăng ký",
        "callbackNo": "Mã Giao dịch (Callback)",
        "paymentAmount": "Số tiền cần thanh toán",
        "balancePayment": "Thanh toán bằng số dư",
        "discountAmount": "Số tiền giảm giá",
        "refundAmount": "Số tiền hoàn lại",
        "deductionAmount": "Số tiền khấu trừ",
        "createdAt": "Ngày tạo",
        "updatedAt": "Ngày cập nhật",
        "commissionStatus": "Trạng thái hoa hồng",
        "commissionAmount": "Hoa hồng kiếm được",
        "actualCommissionAmount": "Hoa hồng thực tế",
        "inviteUser": "Người mời",
        "inviteUserId": "ID Người mời"
      },
      "placeholders": {
        "email": "Nhập email người dùng",
        "plan": "Chọn gói đăng ký",
        "period": "Chọn chu kỳ đăng ký",
        "amount": "Nhập số tiền thanh toán"
      },
      "actions": {
        "cancel": "Hủy",
        "confirm": "Xác nhận"
      },
      "messages": {
        "addOrder": "Thêm đơn hàng thành công",
        "addSuccess": "Thêm thành công"
      }
    }
  },
  "dashboard": {
    "title": "Bảng điều khiển",
    "stats": {
      "newUsers": "Người dùng mới",
      "totalScore": "Tổng điểm",
      "monthlyUpload": "Upload tháng này",
      "vsLastMonth": "so với tháng trước",
      "vsYesterday": "so với hôm qua",
      "todayIncome": "Doanh thu hôm nay",
      "monthlyIncome": "Doanh thu tháng này",
      "totalIncome": "Tổng doanh thu",
      "totalUsers": "Tổng người dùng",
      "activeUsers": "Đang hoạt động: {{count}}",
      "totalOrders": "Tổng đơn hàng",
      "revenue": "Doanh thu",
      "todayRegistered": "Đăng ký hôm nay",
      "monthlyRegistered": "Đăng ký tháng này",
      "onlineUsers": "Đang trực tuyến",
      "pendingTickets": "Ticket chờ",
      "hasPendingTickets": "Có ticket cần trả lời",
      "noPendingTickets": "Không có ticket mở",
      "pendingCommission": "Hoa hồng chờ thanh toán",
      "hasPendingCommission": "Có hoa hồng cần xác nhận",
      "noPendingCommission": "Không có hoa hồng chờ",
      "monthlyNewUsers": "Người dùng mới tháng này",
      "monthlyDownload": "Download tháng này",
      "todayTraffic": "Hôm nay: {{value}}",
      "activeUserTrend": "Xu hướng hoạt động",
      "realtimeUsers": "Người dùng thời gian thực",
      "todayPeak": "Đỉnh hôm nay",
      "vsLastWeek": "so với tuần trước"
    },
    "trafficRank": {
      "nodeTrafficRank": "Xếp hạng Node",
      "userTrafficRank": "Xếp hạng Người dùng",
      "today": "Hôm nay",
      "last7days": "7 ngày",
      "last30days": "30 ngày",
      "customRange": "Tùy chọn",
      "selectTimeRange": "Chọn khoảng thời gian",
      "selectDateRange": "Chọn ngày",
      "currentTraffic": "Traffic kỳ này",
      "previousTraffic": "Traffic kỳ trước",
      "changeRate": "Thay đổi",
      "recordTime": "Thời gian ghi nhận"
    },
    "overview": {
      "title": "Tổng quan doanh thu",
      "thisMonth": "Tháng này",
      "lastMonth": "Tháng trước",
      "to": "đến",
      "selectTimeRange": "Chọn khoảng",
      "selectDate": "Chọn ngày",
      "last7Days": "7 ngày",
      "last30Days": "30 ngày",
      "last90Days": "90 ngày",
      "last180Days": "180 ngày",
      "lastYear": "Năm",
      "customRange": "Tùy chọn",
      "amount":"Giá trị",
      "count": "Số lượng",
      "transactions":"{{count}} giao dịch",
      "orderAmount": "Giá trị đơn hàng",
      "commissionAmount": "Giá trị hoa hồng",
      "orderCount": "Đơn hàng",
      "commissionCount": "Hoa hồng",
      "totalIncome": "Lợi nhuận",
      "totalCommission": "Tổng hoa hồng",
      "totalTransactions": "Giao dịch: {{count}}",
      "avgOrderAmount": "Giá trị TB:",
      "commissionRate": "Tỷ lệ hoa hồng:"
    },
    "traffic": {
      "title": "Xếp hạng Traffic",
      "rank": "Hạng",
      "domain": "Domain",
      "todayTraffic": "Traffic hôm nay",
      "monthlyTraffic": "Traffic tháng này"
    },
    "queue": {
      "title": "Hàng đợi (Queue)",
      "jobDetails": "Chi tiết tác vụ",
      "status": {
        "description": "Trạng thái thực thi hàng đợi hiện tại",
        "running": "Trạng thái",
        "normal": "Bình thường",
        "abnormal": "Lỗi",
        "waitTime": "Chờ: {{seconds}} giây",
        "pending": "Trong hàng đợi",
        "processing": "Đang xử lý",
        "completed": "Hoàn thành",
        "failed": "Thất bại",
        "cancelled": "Đã hủy"
      },
      "details": {
        "description": "Chi tiết kỹ thuật hàng đợi",
        "recentJobs": "Tác vụ gần đây",
        "statisticsPeriod": "Thống kê: {{hours}} giờ",
        "jobsPerMinute": "Tác vụ/phút",
        "maxThroughput": "Tải đỉnh: {{value}}",
        "failedJobs7Days": "Lỗi (7 ngày)",
        "retentionPeriod": "Lưu trữ: {{hours}} giờ",
        "longestRunningQueue": "Hàng đợi dài nhất",
        "activeProcesses": "Tiến trình đang chạy",
        "id": "ID Tác vụ",
        "type": "Loại tác vụ",
        "status": "Trạng thái",
        "progress": "Tiến độ",
        "createdAt": "Tạo lúc",
        "updatedAt": "Cập nhật lúc",
        "error": "Lỗi",
        "data": "Dữ liệu tác vụ",
        "result": "Kết quả",
        "duration": "Thời gian",
        "attempts": "Số lần thử",
        "nextRetry": "Phút đến lần thử lại",
        "failedJobsDetailTitle": "Lỗi hàng đợi",
        "viewFailedJobs": "Lỗi",
        "jobDetailTitle": "Chi tiết thực thi",
        "time": "Thời gian",
        "queue": "Hàng đợi",
        "name": "Tên tác vụ",
        "exception": "Ngoại lệ",
        "noFailedJobs": "Không tìm thấy lỗi",
        "connection": "Kết nối",
        "payload": "Payload",
        "viewDetail": "Chi tiết",
        "action": "Hành động"
      },
      "actions": {
        "retry": "Thử lại",
        "cancel": "Hủy",
        "delete": "Xóa",
        "viewDetails": "Xem thêm"
      },
      "empty": "Không tìm thấy tác vụ",
      "loading": "Đang tải trạng thái hàng đợi...",
      "error": "Lỗi tải trạng thái"
    },
    "common": {
      "refresh": "Làm mới",
      "close": "Đóng",
      "pagination": "Trang {{current}}/{{total}}, tổng {{count}}"
    },
    "search": {
      "placeholder": "Tìm menu và chức năng...",
      "title": "Điều hướng",
      "noResults": "Không tìm thấy",
      "loading": "Đang tìm..."
    }
  },
  "server": {
    "title": "Cấu hình Node",
    "description": "Cài đặt thông số kết nối và đồng bộ node, bao gồm khóa kết nối, khoảng thời gian polling, cân bằng tải và các tham số nâng cao khác.",
    "server_token": {
      "title": "Khóa kết nối",
      "description": "Khóa dùng để giao tiếp giữa Lets VPN và các node nhằm ngăn chặn truy cập trái phép.",
      "placeholder": "Vui lòng nhập khóa kết nối"
    },
    "server_pull_interval": {
      "title": "Khoảng thời gian Pull (Lấy dữ liệu)",
      "description": "Tần suất node lấy dữ liệu từ bảng điều khiển.",
      "placeholder": "Vui lòng nhập khoảng thời gian lấy"
    },
    "server_push_interval": {
      "title": "Khoảng thời gian Push (Gửi dữ liệu)",
      "description": "Tần suất node gửi dữ liệu về bảng điều khiển.",
      "placeholder": "Vui lòng nhập khoảng thời gian gửi"
    },
    "device_limit_mode": {
      "title": "Chế độ giới hạn thiết bị",
      "description": "Ở chế độ mềm, nhiều node từ cùng một IP được tính là một thiết bị.",
      "strict": "Chế độ nghiêm ngặt",
      "relaxed": "Chế độ mềm",
      "placeholder": "Vui lòng chọn chế độ giới hạn"
    },
    "saving": "Đang lưu...",
    "manage": {
      "sync": {
        "label": "Độ phủ cập nhật node",
        "value": "{{synced}} / {{reported}} đã đồng bộ",
        "hint": "Số người dùng đã ở bộ node mới nhất / số người dùng chạy client có báo cáo (tổng {{total}} tài khoản). Phần còn lại sẽ tự đồng bộ ở lần mở ứng dụng hoặc kết nối kế tiếp.",
        "last_change": "Lần thay đổi node gần nhất: {{time}}"
      },
      "title": "Quản lý Node",
      "description": "Quản lý tất cả các node, bao gồm thêm, xóa, sửa và các thao tác khác.",
      "filtered_by_server": "Đang hiển thị node của máy chủ {{server}} (SID:{{id}})",
      "filtered_by_server_description": "Các node mới tại đây có thể được liên kết ngay với máy chủ hiện tại.",
      "add_node_to_server": "Thêm node vào máy chủ này",
      "clear_server_filter": "Xóa bộ lọc máy chủ"
    },
    "columns": {
      "sort": "Sắp xếp",
      "nodeId": "ID Node",
      "show": "Hiển thị",
      "node": "Node",
      "address": "Địa chỉ",
      "onlineUsers": {
        "title": "Người dùng online",
        "tooltip": "Số lượng người dùng online dựa trên tần suất báo cáo của server"
      },
      "rate": {
        "title": "Hệ số",
        "tooltip": "Hệ số tính phí traffic"
      },
      "traffic": {
        "title": "Traffic",
        "tooltip": "Lưu lượng sử dụng của node",
        "total": "Tổng",
        "used": "Đã dùng",
        "percentage": "Sử dụng"
      },
      "groups": {
        "title": "Nhóm truy cập",
        "tooltip": "Các nhóm có thể đăng ký node này",
        "empty": "--"
      },
      "loadStatus": {
        "title": "Tải trọng",
        "tooltip": "Sử dụng tài nguyên máy chủ",
        "noData": "Không có dữ liệu",
        "details": "Chi tiết tải hệ thống",
        "cpu": "Tải CPU",
        "memory": "RAM",
        "swap": "Swap",
        "disk": "Disk",
        "lastUpdate": "Cập nhật cuối",
        "metrics": {
          "title": "Chỉ số",
          "uptime": "Uptime",
          "conns": "Kết nối",
          "speed": "Tốc độ"
        }
      },
      "customId": "Custom ID",
      "originalId": "Original ID",
      "type": "Loại",
      "actions": "Thao tác",
      "copyAddress": "Sao chép địa chỉ",
      "internalPort": "Cổng nội bộ",
      "deployment": {
        "title": "Triển khai",
        "tooltip": "Hiển thị node chạy độc lập hay trên máy chủ, và cho phép thay đổi trực tiếp trong danh sách.",
        "standalone": "Độc lập",
        "standalone_row_hint": "Không liên kết với máy chủ",
        "standalone_description": "Node này hoạt động độc lập và không cần máy chủ lưu trữ.",
        "online": "Server Online",
        "offline": "Server Offline",
        "inactive": "Server không hoạt động",
        "disabled": "Node đã tắt",
        "enabled": "Bật trên Server",
        "enabled_description": "Chỉ các node được bật mới được chạy và đồng bộ bởi server đã chọn.",
        "enabled_standalone_description": "Node độc lập không yêu cầu trạng thái bật của server.",
        "bind_success": "Đã lưu trữ trên {{server}}",
        "standalone_success": "Đã chuyển sang triển khai độc lập",
        "update_success": "Đã cập nhật triển khai",
        "update_error": "Không thể cập nhật triển khai"
      },
      "status": {
        "0": "Chưa chạy",
        "1": "Không sử dụng hoặc Lỗi",
        "2": "Hoạt động bình thường"
      },
      "childNode": "Node con",
      "actions_dropdown": {
        "edit": "Sửa",
        "copy": "Sao chép",
        "delete": {
          "title": "Xác nhận xóa",
          "description": "Hành động này sẽ xóa vĩnh viễn node này và không thể hoàn tác. Bạn có chắc chắn muốn tiếp tục?",
          "confirm": "Xóa"
        },
        "copy_success": "Sao chép thành công",
        
        "delete_success": "Xóa thành công",
        "reset_traffic": {
          "title": "Xác nhận Reset Traffic",
          "description": "Bạn có chắc chắn muốn reset traffic cho node này?",
          "confirm": "Reset Traffic"
        },
        "reset_traffic_success": "Reset Traffic thành công"
      }
    },
    "toolbar": {
      "search": "Tìm node...",
      "type": "Loại",
      "server": "Máy chủ",
      "server_search": "Tìm máy chủ...",
      "server_empty": "Không tìm thấy máy chủ",
      "reset": "Đặt lại",
      "actions": "Thao tác",
      "sort": {
        "tip": "Kéo thả node để sắp xếp, sau đó nhấn lưu",
        "edit": "Sửa thứ tự",
        "save": "Lưu thứ tự",
        "success": "Đã lưu thứ tự sắp xếp"
      },
      "batch_delete": {
        "menu": "Xóa node",
        "button": "Xóa {{count}} cái",
        "title": "Xác nhận xóa",
        "description": "Bạn có chắc chắn muốn xóa {{count}} node? Hành động này không thể đảo ngược.",
        "confirm": "Xác nhận xóa"
      },
      "batch_delete_success": "Đã xóa thành công {{count}} node",
      "batch_delete_error": "Lỗi xóa hàng loạt",
      "batch_show": {
        "menu": "Hiển thị node"
      },
      "batch_show_success": "Đã hiển thị thành công {{count}} node",
      "batch_show_error": "Lỗi hiển thị hàng loạt",
      "batch_hide": {
        "menu": "Ẩn node"
      },
      "batch_hide_success": "Đã ẩn thành công {{count}} node",
      "batch_hide_error": "Lỗi ẩn hàng loạt",
      "batch_enable": {
        "menu": "Bật node"
      },
      "batch_enable_success": "Đã bật thành công {{count}} node",
      "batch_enable_error": "Lỗi bật hàng loạt",
      "batch_disable": {
        "menu": "Tắt node"
      },
      "batch_disable_success": "Đã tắt thành công {{count}} node",
      "batch_disable_error": "Lỗi tắt hàng loạt",
      "batch_reset_traffic": {
        "menu": "Reset Traffic",
        "button": "Reset {{count}} cái",
        "title": "Xác nhận Reset Traffic",
        "description": "Bạn có chắc chắn muốn reset traffic cho {{count}} node? Điều này sẽ làm trắng traffic và gỡ bỏ các khóa.",
        "confirm": "Xác nhận Reset"
      },
      "batch_reset_traffic_success": "Đã reset traffic thành công cho {{count}} node",
      "batch_reset_traffic_error": "Lỗi reset traffic hàng loạt"
    },
    "form": {
      "add_node": "Thêm Node",
      "edit_node": "Sửa Node",
      "new_node": "Node mới",
      "type": {
        "placeholder": "Chọn loại giao thức",
        "select_prompt": "Vui lòng chọn loại giao thức trước",
        "select_error": "Vui lòng chọn loại giao thức"
      },
      "name": {
        "label": "Tên Node",
        "placeholder": "Vui lòng nhập tên node",
        "error": "Vui lòng nhập tên hợp lệ"
      },
      "rate": {
        "label": "Hệ số cơ bản",
        "error": "Hệ số là bắt buộc",
        "error_numeric": "Hệ số phải là số",
        "error_gte_zero": "Hệ số phải lớn hơn hoặc bằng 0",
        "child_node_tooltip": "Hệ số node con được kế thừa từ node cha và không thể thay đổi riêng lẻ",
        "child_node_note": "Hệ số được kế thừa từ cha"
      },
      "traffic_limit": {
        "label": "Giới hạn lưu lượng",
        "placeholder": "0 là không giới hạn",
        "hint": "Thiết lập giới hạn lưu lượng (đơn vị: GB), 0 là không giới hạn",
        "error_numeric": "Giới hạn lưu lượng phải là số",
        "error_gte_zero": "Giới hạn lưu lượng phải lớn hơn hoặc bằng 0"
      },
      "dynamic_rate": {
        "section_title": "Cấu hình hệ số động",
        "enable_label": "Bật hệ số động",
        "enable_description": "Đặt các hệ số khác nhau tùy theo thời gian trong ngày",
        "rules_label": "Quy tắc khung giờ",
        "add_rule": "Thêm quy tắc",
        "rule_title": "Quy tắc {{index}}",
        "start_time": "Giờ bắt đầu",
        "end_time": "Giờ kết thúc",
        "multiplier": "Hệ số",
        "no_rules": "Chưa có quy tắc, nhấn nút trên để thêm",
        "start_time_error": "Giờ bắt đầu là bắt buộc",
        "end_time_error": "Giờ kết thúc là bắt buộc",
        "multiplier_error": "Hệ số là bắt buộc",
        "multiplier_error_numeric": "Hệ số phải là số",
        "multiplier_error_gte_zero": "Hệ số phải >= 0"
      },
      "code": {
        "label": "Custom ID Node",
        "optional": "(Tùy chọn)",
        "placeholder": "Nhập Custom ID Node"
      },
      "tags": {
        "label": "Thẻ Node",
        "placeholder": "Nhấn Enter để thêm"
      },
      "groups": {
        "label": "Nhóm truy cập",
        "add": "Thêm nhóm",
        "placeholder": "Vui lòng chọn nhóm",
        "empty": "Không tìm thấy"
      },
      "machine": {
        "label": "Liên kết với Máy chủ",
        "placeholder": "Chọn máy chủ (tùy chọn)",
        "none": "Độc lập",
        "enabled_hint": "Quản lý node này bằng máy chủ"
      },
      "host": {
        "label": "Địa chỉ Node",
        "placeholder": "Nhập domain hoặc IP",
        "error": "Địa chỉ node là bắt buộc"
      },
      "port": {
        "label": "Cổng kết nối",
        "placeholder": "Cổng cho người dùng",
        "tooltip": "Cổng mà người dùng thực sự kết nối. Nếu dùng transit hoặc tunnel, cổng này có thể khác với cổng server lắng nghe.",
        "sync": "Đồng bộ với cổng server",
        "error": "Cổng kết nối là bắt buộc"
      },
      "server_port": {
        "label": "Cổng Server",
        "placeholder": "Nhập cổng server",
        "error": "Cổng server là bắt buộc",
        "tooltip": "Cổng lắng nghe thực tế trên server.",
        "sync": "Đồng bộ với cổng server"
      },
      "parent": {
        "label": "Node cha",
        "placeholder": "Chọn node cha",
        "none": "Không có"
      },
      "route": {
        "label": "Nhóm Route",
        "placeholder": "Chọn route",
        "empty": "Không tìm thấy"
      },
      "submit": "Gửi",
      "cancel": "Hủy",
      "success": "Gửi thành công"
    },
    "dynamic_form": {
      "multiplex": {
        "enabled": {
          "label": "Multiplex",
          "description": "Truyền nhiều luồng qua một kết nối TCP"
        },
        "protocol": {
          "label": "Giao thức"
        },
        "max_connections": {
          "label": "Max kết nối"
        },
        "min_streams": {
          "label": "Min luồng"
        },
        "padding": {
          "label": "Bật Padding"
        },
        "brutal": {
          "enabled": {
            "label": "TCP Brutal"
          },
          "up_mbps": {
            "label": "Băng thông Upload"
          },
          "down_mbps": {
            "label": "Băng thông Download"
          },
          "description": "TCP Brutal là thuật toán tăng tốc hai chiều. Đặt băng thông 80%-90% so với thực tế. BBR sẽ bị tắt khi bật tính năng này."
        }
      },
      "ech": {
        "description": "Bật Encrypted Client Hello cho các client TLS hỗ trợ. Nếu cấu hình rỗng, sẽ thực hiện truy vấn DNS.",
        "generate": "Tự động tạo khóa ECH",
        "config": {
          "label": "ECH Config (PEM)",
          "placeholder": "Dán cấu hình ECH định dạng PEM, mỗi dòng một chuỗi",
          "description": "Nếu trống, sing-box sẽ cố gắng tải cấu hình ECH từ DNS."
        },
        "config_path": {
          "label": "Đường dẫn file ECH config",
          "placeholder": "/etc/sing-box/ech.pem",
          "description": "Đường dẫn đến file cấu hình ECH định dạng PEM."
        },
        "query_server_name": {
          "label": "Tên server cho truy vấn ECH",
          "placeholder": "Tùy chọn thay thế domain cho truy vấn HTTPS",
          "description": "Ghi đè domain cho truy vấn HTTPS ECH. Mặc định dùng server_name."
        },
        "key": {
          "label": "ECH Key",
          "placeholder": "Dán nội dung ECH key nếu backend yêu cầu",
          "description": "Nội dung ECH key tùy chọn cho backend."
        },
        "key_path": {
          "label": "Đường dẫn ECH key",
          "placeholder": "/etc/sing-box/ech.key",
          "description": "Đường dẫn đến file ECH key nếu backend yêu cầu."
        }
      },
      "anytls": {
        "tls": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Dùng để xác thực chứng chỉ khi domain khác với địa chỉ node"
          },
          "allow_insecure": "Cho phép Không an toàn"
        },
        "padding_scheme": {
          "description": "Mỗi dòng là một quy tắc padding, định dạng: stop=8, 0=30-30",
          "label": "Sơ đồ Padding",
          "placeholder": "Nhập quy tắc padding",
          "use_default": "Dùng mặc định",
          "edit_btn": "Sửa sơ đồ Padding",
          "configured": "Đã cấu hình {{count}} quy tắc",
          "not_configured": "Chưa cấu hình"
        }
      },
      "shadowsocks": {
        "cipher": {
          "label": "Phương thức mã hóa",
          "placeholder": "Chọn phương thức mã hóa",
          "search_placeholder": "Tìm hoặc nhập phương thức tùy chỉnh...",
          "description": "Chọn phương thức mã hóa có sẵn hoặc nhập tùy chỉnh",
          "preset_group": "Phương thức có sẵn",
          "custom_group": "Phương thức tùy chỉnh",
          "current_value": "Giá trị hiện tại",
          "use_custom": "Sử dụng",
          "no_results": "Không tìm thấy phương thức mã hóa",
          "custom_hint": "Bạn có thể nhập trực tiếp phương thức mã hóa, ví dụ: aes-256-cfb",
          "custom_label": "Tùy chỉnh"
        },
        "plugin": {
          "label": "Plugin",
          "placeholder": "Chọn plugin",
          "obfs_hint": "Gợi ý: Định dạng cấu hình như obfs=http;obfs-host=www.bing.com;path=/",
          "v2ray_hint": "Gợi ý: Định dạng chế độ WebSocket là mode=websocket;host=mydomain.me;path=/;tls=true, chế độ QUIC là mode=quic;host=mydomain.me",
          "gost_hint": "Gợi ý: Định dạng cấu hình như mode=websocket;host=mydomain.me;path=/;tls=true",
          "shadow_tls_hint": "Gợi ý: Định dạng cấu hình như host=cloud.tencent.com;password=auth_password;version=3",
          "restls_hint": "Gợi ý: Định dạng cấu hình như host=www.microsoft.com;password=auth_password;version-hint=tls13;restls-script=300?100<1,400~100",
          "kcptun_hint": "Gợi ý: Định dạng cấu hình như key=psk;crypt=aes-128-gcm;mode=fast;mtu=1350"
        },
        "plugin_opts": {
          "label": "Tùy chọn Plugin",
          "description": "Nhập tùy chọn plugin theo định dạng key=value;key2=value2",
          "placeholder": "Ví dụ: mode=tls;host=bing.com"
        },
        "client_fingerprint": "Client Fingerprint",
        "client_fingerprint_placeholder": "Chọn Client Fingerprint",
        "client_fingerprint_description": "Giả mạo fingerprint client để giảm nguy cơ bị phát hiện",
        "obfs": {
          "label": "Obfuscation",
          "placeholder": "Chọn phương thức obfuscation",
          "none": "Không",
          "http": "HTTP"
        },
        "obfs_settings": {
          "path": "Path",
          "host": "Host"
        },
        "cert_config": {
          "tab": "TLS Cert",
          "cert_mode": {
            "label": "Chế độ Cert",
            "description": "Chọn cách quản lý chứng chỉ",
            "self_description": "Self-signed: Valid 10 năm, tự động tạo bởi node",
            "http_description": "HTTP-01: Cổng 80 phải có thể truy cập",
            "dns_description": "DNS-01: Xác thực qua bản ghi DNS, hỗ trợ wildcard",
            "content_description": "Content: Đẩy file cert thô lên node"
          },
          "domain": {
            "label": "Domain",
            "placeholder": "example.com"
          },
          "email": {
            "label": "Email thông báo",
            "placeholder": "admin@example.com"
          },
          "http_port": {
            "label": "Cổng Challenge",
            "description": "Cổng challenge ACME (mặc định 80)"
          },
          "dns_provider": {
            "label": "Nhà cung cấp DNS",
            "doc_link": "Hướng dẫn cấu hình nhà cung cấp DNS"
          },
          "dns_env": {
            "label": "Biến môi trường (API Keys)",
            "description_short": "Mỗi dòng một KEY=VALUE"
          },
          "cert_content": {
            "label": "Nội dung Public Key"
          },
          "key_content": {
            "label": "Nội dung Private Key"
          },
          "none_desc": "Cấu hình TLS đã tắt"
        }
      },
      "vmess": {
        "tls": {
          "label": "TLS",
          "placeholder": "Vui lòng chọn bảo mật",
          "disabled": "Tắt",
          "enabled": "Bật"
        },
        "tls_settings": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Để trống nếu không sử dụng"
          },
          "allow_insecure": "Cho phép Không an toàn?"
        },
        "network": {
          "label": "Giao thức truyền tải",
          "placeholder": "Chọn giao thức truyền tải"
        }
      },
      "trojan": {
        "server_name": {
          "label": "Server Name Indication (SNI)",
          "placeholder": "Dùng để xác thực chứng chỉ khi địa chỉ node khác với chứng chỉ"
        },
        "allow_insecure": "Cho phép Không an toàn?",
        "reality_settings": {
          "server_name": {
            "label": "Trang đích (dest)",
            "placeholder": "ví dụ: example.com"
          },
          "server_port": {
            "label": "Cổng",
            "placeholder": "ví dụ: 443"
          },
          "allow_insecure": "Cho phép Không an toàn?",
          "private_key": {
            "label": "Private Key"
          },
          "public_key": {
            "label": "Public Key"
          },
          "short_id": {
            "label": "Short ID",
            "placeholder": "Tùy chọn, độ dài phải chẵn, tối đa 16 ký tự",
            "description": "Danh sách shortIds có sẵn cho client, dùng để phân biệt các client khác nhau, sử dụng ký tự hex 0-f",
            "generate": "Tạo Short ID",
            "success": "Tạo Short ID thành công"
          },
          "key_pair": {
            "generate": "Tạo cặp khóa",
            "success": "Tạo cặp khóa thành công",
            "error": "Không thể tạo cặp khóa"
          }
        },
        "network": {
          "label": "Giao thức truyền tải",
          "placeholder": "Chọn giao thức truyền tải"
        }
      },
      "hysteria": {
        "version": {
          "label": "Phiên bản giao thức",
          "placeholder": "Phiên bản giao thức"
        },
        "alpn": {
          "label": "ALPN",
          "placeholder": "ALPN"
        },
        "obfs": {
          "label": "Obfuscation",
          "type": {
            "label": "Implementation Obfuscation",
            "placeholder": "Chọn implementation obfuscation",
            "salamander": "Salamander"
          },
          "password": {
            "label": "Mật khẩu Obfuscation",
            "placeholder": "Vui lòng nhập mật khẩu obfuscation",
            "generate_success": "Tạo mật khẩu obfuscation thành công"
          }
        },
        "tls": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Dùng để xác thực chứng chỉ khi địa chỉ node khác với chứng chỉ"
          },
          "allow_insecure": "Cho phép Không an toàn?"
        },
        "bandwidth": {
          "up": {
            "label": "Băng thông Upload",
            "placeholder": "Vui lòng nhập băng thông upload",
            "suffix": "Mbps",
            "bbr_tip": ", để trống để dùng BBR"
          },
          "down": {
            "label": "Băng thông Download",
            "placeholder": "Vui lòng nhập băng thông download",
            "suffix": "Mbps",
            "bbr_tip": ", để trống để dùng BBR"
          }
        }
      },
      "vless": {
        "tls": {
          "label": "Bảo mật",
          "placeholder": "Vui lòng chọn bảo mật",
          "none": "Không",
          "tls": "TLS",
          "reality": "Reality"
        },
        "tls_settings": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Để trống nếu không sử dụng"
          },
          "allow_insecure": "Cho phép Không an toàn?"
        },
        "reality_settings": {
          "server_name": {
            "label": "Trang đích (dest)",
            "placeholder": "ví dụ: example.com"
          },
          "server_port": {
            "label": "Cổng",
            "placeholder": "ví dụ: 443"
          },
          "allow_insecure": "Cho phép Không an toàn?",
          "private_key": {
            "label": "Private Key"
          },
          "public_key": {
            "label": "Public Key"
          },
          "short_id": {
            "label": "Short ID",
            "placeholder": "Tùy chọn, độ dài phải chẵn, tối đa 16 ký tự",
            "description": "Danh sách shortIds có sẵn cho client, dùng để phân biệt các client khác nhau, sử dụng ký tự hex 0-f",
            "generate": "Tạo Short ID",
            "success": "Tạo Short ID thành công"
          },
          "key_pair": {
            "generate": "Tạo cặp khóa",
            "success": "Tạo cặp khóa thành công",
            "error": "Không thể tạo cặp khóa"
          }
        },
        "network": {
          "label": "Giao thức truyền tải",
          "placeholder": "Chọn giao thức truyền tải"
        },
        "flow": {
          "label": "Kiểm soát luồng (Flow)",
          "placeholder": "Chọn kiểm soát luồng"
        },
        "encryption": {
          "label": "Mã hóa VLESS",
          "description": "Bật mã hóa VLESS",
          "server_label": "giải mã",
          "server_placeholder": "./xray vlessenc",
          "server_description": "",
          "client_label": "mã hóa",
          "client_placeholder": "./xray vlessenc",
          "client_description": "",
          "generate_hint": "./xray vlessenc"
        }
      },
      "tuic": {
        "version": {
          "label": "Phiên bản giao thức",
          "placeholder": "Chọn phiên bản TUIC"
        },
        "password": {
          "label": "Mật khẩu",
          "placeholder": "Nhập mật khẩu",
          "generate_success": "Tạo mật khẩu thành công"
        },
        "congestion_control": {
          "label": "Kiểm soát tắc nghẽn",
          "placeholder": "Chọn thuật toán kiểm soát tắc nghẽn"
        },
        "udp_relay_mode": {
          "label": "Chế độ UDP Relay",
          "placeholder": "Chọn chế độ UDP Relay"
        },
        "tls": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Dùng để xác thực chứng chỉ khi domain khác với địa chỉ node"
          },
          "allow_insecure": "Cho phép Không an toàn?",
          "alpn": {
            "label": "ALPN",
            "placeholder": "Chọn giao thức ALPN",
            "empty": "Không có giao thức ALPN"
          }
        }
      },
      "socks": {
        "version": {
          "label": "Phiên bản giao thức",
          "placeholder": "Chọn phiên bản SOCKS"
        },
        "tls": {
          "label": "TLS",
          "placeholder": "Vui lòng chọn bảo mật",
          "disabled": "Tắt",
          "enabled": "Bật"
        },
        "tls_settings": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Để trống nếu không sử dụng"
          },
          "allow_insecure": "Cho phép Không an toàn?"
        },
        "network": {
          "label": "Giao thức truyền tải",
          "placeholder": "Chọn giao thức truyền tải"
        }
      },
      "naive": {
        "tls_settings": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Dùng để xác thực chứng chỉ khi domain khác với địa chỉ node"
          },
          "allow_insecure": "Cho phép Không an toàn"
        },
        "tls": {
          "label": "TLS",
          "placeholder": "Vui lòng chọn bảo mật",
          "disabled": "Tắt",
          "enabled": "Bật",
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Dùng để xác thực chứng chỉ khi domain khác với địa chỉ node"
          },
          "allow_insecure": "Cho phép Không an toàn"
        }
      },
      "http": {
        "tls_settings": {
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Dùng để xác thực chứng chỉ khi domain khác với địa chỉ node"
          },
          "allow_insecure": "Cho phép Không an toàn"
        },
        "tls": {
          "label": "TLS",
          "placeholder": "Vui lòng chọn bảo mật",
          "disabled": "Tắt",
          "enabled": "Bật",
          "server_name": {
            "label": "Server Name Indication (SNI)",
            "placeholder": "Dùng để xác thực chứng chỉ khi domain khác với địa chỉ node"
          },
          "allow_insecure": "Cho phép Không an toàn"
        }
      },
      "mieru": {
        "transport": {
          "label": "Giao thức truyền tải",
          "placeholder": "Chọn giao thức truyền tải"
        },
        "traffic_pattern": {
          "label": "Mẫu Traffic (Base64)",
          "placeholder": "Nhập chuỗi Base64 để tinh chỉnh mạng",
          "success": "Đã tạo mẫu traffic"
        }
      },
      "advanced": {
        "trigger_label": "Cấu hình nâng cao",
        "dialog_title": "Cấu hình giao thức nâng cao",
        "tls_tab": "TLS",
        "route_tab": "Định tuyến",
        "multiplex_tab": "Multiplex"
      }
    },
    "network_settings": {
      "edit_protocol": "Sửa giao thức",
      "edit_protocol_config": "Sửa cấu hình giao thức",
      "use_template": "Dùng mẫu {{template}}",
      "json_config_placeholder": "Vui lòng nhập cấu hình JSON",
      "json_config_placeholder_with_template": "Vui lòng nhập cấu hình JSON hoặc chọn mẫu ở trên",
      "validation": {
        "must_be_object": "Cấu hình phải là đối tượng JSON",
        "invalid_json": "Định dạng JSON không hợp lệ"
      },
      "errors": {
        "save_failed": "Có lỗi xảy ra khi lưu"
      }
    },
    "common": {
      "cancel": "Hủy",
      "confirm": "Xác nhận"
    }
  },
  "search": {
    "placeholder": "Tìm menu và chức năng...",
    "title": "Điều hướng menu",
    "noResults": "Không tìm thấy kết quả",
    "shortcut": {
      "label": "Tìm kiếm",
      "key": "⌘K"
    }
  },
  "coupon": {
    "title": "Quản lý Mã giảm giá",
    "description": "Tại đây bạn có thể quản lý mã giảm giá, bao gồm thêm, xem và xóa.",
    "table": {
      "columns": {
        "id": "ID",
        "show": "Bật",
        "name": "Tên",
        "type": "Loại",
        "code": "Mã",
        "limitUse": "Còn lại",
        "limitUseWithUser": "Dùng/người",
        "validity": "Thời hạn",
        "actions": "Thao tác"
      },
      "validity": {
        "expired": "Hết hạn {{days}} ngày trước",
        "notStarted": "Bắt đầu sau {{days}} ngày",
        "remaining": "Còn {{days}} ngày",
        "startTime": "Bắt đầu",
        "endTime": "Kết thúc",
        "unlimited": "Vô hạn",
        "noLimit": "Không giới hạn"
      },
      "actions": {
        "edit": "Sửa",
        "delete": "Xóa",
        "deleteConfirm": {
          "title": "Xác nhận xóa",
          "description": "Hành động này sẽ xóa vĩnh viễn mã giảm giá này. Bạn có chắc chắn muốn tiếp tục?",
          "confirmText": "Xóa"
        }
      },
      "toolbar": {
        "search": "Tìm mã giảm giá...",
        "type": "Loại",
        "reset": "Đặt lại",
        "types": {
          "1": "Số tiền cố định",
          "2": "Phần trăm"
        }
      }
    },
    "form": {
      "add": "Thêm mã giảm giá",
      "edit": "Sửa mã giảm giá",
      "name": {
        "label": "Tên mã giảm giá",
        "placeholder": "Nhập tên mã giảm giá",
        "required": "Vui lòng nhập tên mã giảm giá"
      },
      "type": {
        "label": "Loại và giá trị mã",
        "placeholder": "Chọn loại mã giảm giá"
      },
      "value": {
        "placeholder": "Nhập giá trị"
      },
      "validity": {
        "label": "Thời hạn",
        "to": "đến",
        "endTimeError": "Thời gian kết thúc phải sau thời gian bắt đầu"
      },
      "limitUse": {
        "label": "Tổng số lần sử dụng tối đa",
        "placeholder": "Để trống để không giới hạn",
        "description": "Đặt tổng số lần mã này có thể được sử dụng"
      },
      "limitUseWithUser": {
        "label": "Giới hạn mỗi người dùng",
        "placeholder": "Để trống để không giới hạn",
        "description": "Giới hạn số lần mỗi người dùng có thể sử dụng mã này"
      },
      "limitPeriod": {
        "label": "Chu kỳ đăng ký",
        "placeholder": "Giới hạn chu kỳ cụ thể, để trống cho tất cả",
        "description": "Chọn chu kỳ đăng ký áp dụng cho mã này",
        "empty": "Không tìm thấy chu kỳ"
      },
      "limitPlan": {
        "label": "Gói đăng ký",
        "placeholder": "Giới hạn gói cụ thể, để trống cho tất cả",
        "description": "Chọn gói đăng ký áp dụng cho mã này, để trống cho tất cả",
        "empty": "Không tìm thấy gói"
      },
      "code": {
        "label": "Mã tùy chỉnh",
        "placeholder": "Để trống để tự động tạo",
        "description": "Bạn có thể đặt mã riêng hoặc để trống"
      },
      "generateCount": {
        "label": "Số lượng tạo",
        "placeholder": "Số lượng mã cần tạo, để trống để tạo 1",
        "description": "Tạo nhiều mã giảm giá cùng lúc"
      },
      "submit": {
        "saving": "Đang lưu...",
        "save": "Lưu"
      },
      "error": {
        "saveFailed": "Không thể lưu mã giảm giá"
      },
      "timeRange": {
        "quickSet": "Cài đặt nhanh",
        "presets": {
          "1week": "1 Tuần",
          "2weeks": "2 Tuần",
          "1month": "1 Tháng",
          "3months": "3 Tháng",
          "6months": "6 Tháng",
          "1year": "1 Năm"
        }
      }
    },
    "period": {
      "monthly": "1 Tháng",
      "quarterly": "3 Tháng",
      "half_yearly": "6 Tháng",
      "yearly": "1 năm",
      "two_yearly": "2 năm",
      "three_yearly": "3 năm",
      "onetime": "Vĩnh viễn",
      "reset_traffic": "Đặt lại dung lượng"
    }
  },
  "traffic": {
    "trafficRecord": {
      "title": "Lịch sử sử dụng Traffic",
      "time": "Thời gian",
      "upload": "Upload",
      "download": "Download",
      "rate": "Hệ số",
      "total": "Tổng",
      "noRecords": "Không có bản ghi",
      "perPage": "Hiển thị",
      "records": "bản ghi",
      "page": "Trang {{current}} / {{total}}",
      "multiplier": "{{value}}x"
    }
  },
  "route": {
    "title": "Quản lý Route",
    "description": "Quản lý các nhóm route, bao gồm thêm, xóa và sửa.",
    "columns": {
      "id": "ID Nhóm",
      "remarks": "Ghi chú",
      "action": "Hành động",
      "actions": "Thao tác",
      "matchRules": "Số luật: {{count}}",
      "action_value": {
        "title": "Giá trị hành động",
        "dns": "DNS: {{value}}",
        "proxy": "Proxy ({{value}})",
        "block": "Chặn truy cập",
        "direct": "Kết nối trực tiếp"
      }
    },
    "actions": {
      "dns": "Giải quyết qua DNS",
      "block": "Chặn",
      "direct": "Trực tiếp",
      "proxy": "Proxy"
    },
    "form": {
      "add": "Thêm Route",
      "edit": "Sửa Route",
      "create": "Tạo Route",
      "remarks": "Ghi chú",
      "remarksPlaceholder": "Nhập ghi chú",
      "match": "Luật khớp",
      "matchPlaceholder": "example.com\n*.example.com",
      "action": "Hành động",
      "actionPlaceholder": "Chọn hành động",
      "dns": "Server DNS",
      "dnsPlaceholder": "Nhập server DNS",
      "proxy": "Outbound Tag",
      "proxyPlaceholder": "Nhập Outbound Tag",
      "cancel": "Hủy",
      "submit": "Gửi",
      "validation": {
        "remarks": "Nhập ghi chú hợp lệ"
      }
    },
    "toolbar": {
      "searchPlaceholder": "Tìm route...",
      "reset": "Đặt lại"
    },
    "messages": {
      "deleteConfirm": "Xác nhận xóa",
      "deleteDescription": "Hành động này sẽ xóa vĩnh viễn nhóm route này. Tiếp tục?",
      "deleteButton": "Xóa",
      "deleteSuccess": "Xóa thành công",
      "createSuccess": "Tạo thành công",
      "updateSuccess": "Cập nhật thành công"
    }
  },
  "settings": {
    "title": "Cài đặt Hệ thống",
    "description": "Quản lý cấu hình cơ bản của hệ thống, bao gồm trang web, bảo mật, đăng ký, chương trình giới thiệu, node và thông báo.",
    "server": {
      "title": "Cấu hình Server",
      "description": "Cài đặt kết nối và đồng bộ node, bao gồm khóa kết nối, khoảng polling, cân bằng tải và các tham số khác.",
      "server_token": {
        "title": "Token kết nối",
        "placeholder": "Nhập token kết nối",
        "description": "Dùng để xác thực giữa server và bảng điều khiển",
        "generate_tooltip": "Nhấn để tạo token ngẫu nhiên"
      },
      "server_pull_interval": {
        "title": "Khoảng Pull (Lấy dữ liệu)",
        "description": "Tần suất node yêu cầu dữ liệu từ bảng điều khiển.",
        "placeholder": "Nhập khoảng thời gian"
      },
      "server_push_interval": {
        "title": "Khoảng Push (Gửi dữ liệu)",
        "description": "Tần suất node gửi thống kê về bảng điều khiển.",
        "placeholder": "Nhập khoảng thời gian"
      },
      "device_limit_mode": {
        "title": "Chế độ giới hạn thiết bị",
        "description": "Ở chế độ mềm, nhiều node từ cùng một IP được tính là một thiết bị.",
        "strict": "Nghiêm ngặt",
        "relaxed": "Mềm",
        "placeholder": "Chọn chế độ giới hạn"
      },
      "server_ws_enable": {
        "title": "Bật kết nối WebSocket",
        "description": "Khi bật, node sẽ giao tiếp với bảng điều khiển qua WebSocket để giảm độ trễ.",
        "supported_clients": "Client hỗ trợ WebSocket: Xboard Node"
      },
      "server_ws_url": {
        "title": "URL WebSocket",
        "description": "Địa chỉ WebSocket để node kết nối. Để trống để dùng URL trang web.",
        "placeholder": "Để trống để dùng URL trang web"
      },
      "saving": "Đang lưu..."
    },
    "invite": {
      "title": "Chương trình Giới thiệu",
      "description": "Cài đặt đăng ký qua lời mời và hoa hồng.",
      "invite_force": {
        "title": "Bắt buộc mời",
        "description": "Nếu bật, chỉ có thể đăng ký bằng mã mời."
      },
      "invite_commission": {
        "title": "Phần trăm hoa hồng",
        "description": "Phần trăm hoa hồng mặc định. Có thể cấu hình riêng cho từng người dùng.",
        "placeholder": "Nhập phần trăm"
      },
      "invite_gen_limit": {
        "title": "Giới hạn tạo mã",
        "description": "Số lượng mã tối đa mà người dùng có thể tạo.",
        "placeholder": "Nhập giới hạn"
      },
      "invite_never_expire": {
        "title": "Mã không hết hạn",
        "description": "Nếu bật, mã không bị vô hiệu sau khi sử dụng."
      },
      "commission_first_time": {
        "title": "Hoa hồng lần đầu",
        "description": "Hoa hồng chỉ được tính cho lần thanh toán đầu tiên của người được mời."
      },
      "commission_auto_check": {
        "title": "Tự động xác nhận",
        "description": "Hoa hồng được tự động xác nhận sau 3 ngày kể từ khi đơn hàng hoàn tất."
      },
      "commission_withdraw_limit": {
        "title": "Số tiền rút tối thiểu",
        "description": "Yêu cầu rút dưới mức này sẽ không được chấp nhận.",
        "placeholder": "Nhập số tiền"
      },
      "commission_withdraw_method": {
        "title": "Phương thức rút tiền",
        "description": "Các cách rút tiền được hỗ trợ, phân cách bằng dấu phẩy.",
        "placeholder": "Nhập phương thức (vd: USDT, Thẻ)"
      },
      "withdraw_close": {
        "title": "Tắt rút tiền",
        "description": "Nếu bật, người dùng không thể yêu cầu rút, hoa hồng sẽ cộng thẳng vào số dư."
      },
      "commission_distribution": {
        "title": "Hệ thống 3 cấp",
        "description": "Phân phối hoa hồng cho 3 cấp người mời. Tổng không được vượt quá 100%.",
        "l1": "Cấp 1 (%)",
        "l2": "Cấp 2 (%)",
        "l3": "Cấp 3 (%)",
        "placeholder": "Nhập phần trăm"
      },
      "saving": "Đang lưu..."
    },
    "site": {
      "title": "Cài đặt Trang web",
      "description": "Thông tin cơ bản về trang web: tên, mô tả, tiền tệ và các tham số khác.",
      "form": {
        "siteName": {
          "label": "Tên trang web",
          "placeholder": "Nhập tên",
          "description": "Hiển thị trên tiêu đề và giao diện."
        },
        "siteDescription": {
          "label": "Mô tả trang web",
          "placeholder": "Nhập mô tả",
          "description": "Dùng cho SEO và phụ đề."
        },
        "siteUrl": {
          "label": "URL trang web",
          "placeholder": "Vd: https://domain.com (không có / ở cuối)",
          "description": "Dùng trong liên kết, email và API."
        },
        "forceHttps": {
          "label": "Bắt buộc HTTPS",
          "description": "Bật nếu trang web đằng sau proxy/CDN dùng HTTPS nhưng server chạy HTTP."
        },
        "logo": {
          "label": "Logo (URL)",
          "placeholder": "URL logo của bạn",
          "description": "Liên kết đến hình ảnh logo."
        },
        "subscribeUrl": {
          "label": "URL Đăng ký (tùy chỉnh)",
          "placeholder": "Phân cách bằng ',', để trống để dùng URL trang web",
          "description": "URL dùng để tạo liên kết đăng ký."
        },
        "tosUrl": {
          "label": "URL Điều khoản (TOS)",
          "placeholder": "Liên kết đến quy định dịch vụ",
          "description": "Liên kết đến trang quy định sử dụng."
        },
        "stopRegister": {
          "label": "Tắt đăng ký",
          "description": "Ngăn chặn đăng ký người dùng mới."
        },
        "ticketMustWaitReply": {
          "label": "Chờ phản hồi Ticket",
          "description": "Người dùng không thể gửi tin nhắn mới nếu admin chưa trả lời."
        },
        "tryOut": {
          "label": "Dùng thử",
          "placeholder": "Tắt",
          "description": "Chọn gói dùng thử khi đăng ký.",
          "duration": {
            "label": "Thời lượng (giờ)",
            "placeholder": "0",
            "description": "Thời gian hiệu lực của gói dùng thử."
          }
        },
        "currency": {
          "label": "Mã tiền tệ",
          "placeholder": "VND",
          "description": "Mã hiển thị (ảnh hưởng toàn bộ giao diện)."
        },
        "currencySymbol": {
          "label": "Ký hiệu tiền tệ",
          "placeholder": "₫",
          "description": "Ký hiệu hiển thị (ảnh hưởng toàn bộ giao diện)."
        }
      }
    },
    "safe": {
      "title": "Bảo mật",
      "description": "Cài đặt bảo vệ: captcha, giới hạn mật khẩu, whitelist Email và các tham số khác.",
      "form": {
        "emailVerify": {
          "label": "Xác thực Email",
          "description": "Bắt buộc xác nhận email khi đăng ký."
        },
        "gmailLimit": {
          "label": "Cấm alias Gmail",
          "description": "Cấm đăng ký sử dụng '+' trong địa chỉ Gmail."
        },
        "safeMode": {
          "label": "Chế độ an toàn Domain",
          "description": "Chỉ truy cập trang web qua URL chỉ định (các URL khác trả về 403)."
        },
        "securePath": {
          "label": "Đường dẫn Admin",
          "placeholder": "admin",
          "description": "Thay đổi địa chỉ đăng nhập admin mặc định."
        },
        "emailWhitelist": {
          "label": "Whitelist Domain Email",
          "description": "Chỉ cho phép đăng ký với các domain được liệt kê.",
          "suffixes": {
            "label": "Domain Email",
            "placeholder": "Mỗi dòng một domain (vd: gmail.com)",
            "description": "Danh sách các dịch vụ email được phép."
          }
        },
        "captcha": {
          "enable": {
            "label": "Bật Captcha",
            "description": "Kiểm tra người dùng khi đăng ký."
          },
          "type": {
            "label": "Loại Captcha",
            "description": "Chọn dịch vụ xác minh",
            "options": {
              "recaptcha": "Google reCAPTCHA v2",
              "recaptcha-v3": "Google reCAPTCHA v3",
              "turnstile": "Cloudflare Turnstile"
            }
          },
          "recaptcha": {
            "key": {
              "label": "Secret Key",
              "placeholder": "Nhập Secret Key",
              "description": "Khóa bí mật reCAPTCHA của bạn"
            },
            "siteKey": {
              "label": "Site Key",
              "placeholder": "Nhập Site Key",
              "description": "Khóa công khai reCAPTCHA của bạn"
            }
          },
          "recaptcha_v3": {
            "secretKey": {
              "label": "v3 Secret Key",
              "placeholder": "Nhập v3 Secret Key",
              "description": "Khóa bí mật v3"
            },
            "siteKey": {
              "label": "v3 Site Key",
              "placeholder": "Nhập v3 Site Key",
              "description": "Khóa công khai v3"
            },
            "scoreThreshold": {
              "label": "Ngưỡng điểm (Score)",
              "placeholder": "0.5",
              "description": "Ngưỡng (0-1), càng cao càng có khả năng là người."
            }
          },
          "turnstile": {
            "secretKey": {
              "label": "Turnstile Secret Key",
              "placeholder": "Nhập khóa bí mật",
              "description": "Khóa từ Cloudflare Turnstile"
            },
            "siteKey": {
              "label": "Turnstile Site Key",
              "placeholder": "Nhập Site Key",
              "description": "Khóa công khai Turnstile"
            }
          }
        },
        "registerLimit": {
          "enable": {
            "label": "Giới hạn đăng ký theo IP",
            "description": "Giới hạn số tài khoản từ một địa chỉ IP."
          },
          "count": {
            "label": "Số lần đăng ký",
            "placeholder": "Số tài khoản tối đa",
            "description": "Số lần có thể đăng ký từ một IP."
          },
          "expire": {
            "label": "Thời lượng (phút)",
            "placeholder": "Thời gian khóa",
            "description": "Giới hạn sẽ reset sau bao nhiêu phút."
          }
        },
        "passwordLimit": {
          "enable": {
            "label": "Giới hạn thử mật khẩu",
            "description": "Bảo vệ chống brute-force mật khẩu."
          },
          "count": {
            "label": "Số lần thử",
            "placeholder": "Số lần thử tối đa",
            "description": "Số lần nhập sai mật khẩu cho phép."
          },
          "expire": {
            "label": "Thời gian khóa (phút)",
            "placeholder": "Thời gian ban",
            "description": "Khóa đăng nhập trong bao nhiêu phút."
          }
        }
      }
    },
    "subscribe": {
      "title": "Cài đặt Đăng ký",
      "description": "Quản lý định dạng liên kết, tần suất cập nhật, thống kê traffic và các tham số khác của đăng ký.",
      "plan_change_enable": {
        "title": "Cho phép đổi gói",
        "description": "Cho phép người dùng tự chuyển sang gói khác."
      },
      "reset_traffic_method": {
        "title": "Phương pháp Reset Traffic toàn cục",
        "description": "Cài đặt reset mặc định (ngày 1 hàng tháng). Có thể ghi đè trong từng gói.",
        "options": {
          "monthly_first": "Ngày 1 hàng tháng",
          "monthly_reset": "Hàng tháng (ngày mua)",
          "no_reset": "Không reset",
          "yearly_first": "Ngày 1 tháng 1 hàng năm",
          "yearly_reset": "Hàng năm (ngày mua)"
        }
      },
      "surplus_enable": {
        "title": "Chế độ bù trừ",
        "description": "Khi đổi gói, số dư từ gói cũ sẽ được tính vào gói mới."
      },
      "new_order_event": {
        "title": "Sự kiện khi mua mới",
        "description": "Thực hiện gì khi người dùng mua gói mới.",
        "options": {
          "no_action": "Không làm gì",
          "reset_traffic": "Reset traffic đã dùng"
        }
      },
      "renew_order_event": {
        "title": "Sự kiện khi gia hạn",
        "description": "Thực hiện gì khi thanh toán gia hạn.",
        "options": {
          "no_action": "Không làm gì",
          "reset_traffic": "Reset traffic đã dùng"
        }
      },
      "change_order_event": {
        "title": "Sự kiện khi đổi gói",
        "description": "Thực hiện gì khi chuyển sang gói khác.",
        "options": {
          "no_action": "Không làm gì",
          "reset_traffic": "Reset traffic đã dùng"
        }
      },
      "subscribe_path": {
        "title": "Đường dẫn Đăng ký",
        "description": "Định dạng liên kết. Thay đổi sẽ ảnh hưởng đến tất cả liên kết hiện có.",
        "current_format": "Định dạng hiện tại: {path}/xxxxxxxxxx",
        "restart_tip": "Có thể cần khởi động lại dịch vụ để áp dụng."
      },
      "show_info_to_server": {
        "title": "Thông tin trong đăng ký (client)",
        "description": "Hiển thị thông tin traffic và hạn dùng trong ứng dụng người dùng."
      },
      "show_protocol_to_server": {
        "title": "Giao thức trong tên node",
        "description": "Thêm tên giao thức vào node (vd: [Hy2] Hong Kong)."
      },
      "saving": "Đang lưu...",
      "plan": {
        "title": "Gói đăng ký",
        "add": "Thêm",
        "search": "Tìm...",
        "sort": {
          "edit": "Sắp xếp",
          "save": "Lưu"
        },
        "columns": {
          "id": "ID",
          "show": "Hiển thị",
          "sell": "Bán",
          "renew": "Gia hạn",
          "renew_tooltip": "Có thể gia hạn nếu gói ngừng bán không",
          "name": "Tên",
          "stats": "Thống kê",
          "group": "Nhóm",
          "price": "Giá",
          "actions": "Thao tác",
          "edit": "Sửa",
          "delete": "Xóa",
          "delete_confirm": {
            "title": "Xóa",
            "description": "Không thể hoàn tác. Tiếp tục?",
            "success": "Đã xóa"
          },
          "price_period": {
            "monthly": "1 Tháng",
            "quarterly": "3 Tháng",
            "half_yearly": "6 Tháng",
            "yearly": "1 năm",
            "two_yearly": "2 năm",
            "three_yearly": "3 năm",
            "onetime": "Vĩnh viễn",
            "reset_traffic": "Đặt lại dung lượng",
            "unit": {
              "month": "/tháng",
              "quarter": "/quý",
              "half_year": "/6tháng",
              "year": "/năm",
              "two_year": "/2năm",
              "three_year": "/3năm",
              "times": "/lần"
            }
          }
        },
        "form": {
          "add_title": "Thêm gói",
          "edit_title": "Sửa gói",
          "name": {
            "label": "Tên",
            "placeholder": "Nhập tên"
          },
          "group": {
            "label": "Nhóm truy cập",
            "placeholder": "Chọn nhóm",
            "add": "Thêm"
          },
          "transfer": {
            "label": "Traffic",
            "placeholder": "Dung lượng traffic",
            "unit": "GB"
          },
          "speed": {
            "label": "Giới hạn tốc độ",
            "placeholder": "Tốc độ tối đa",
            "unit": "Mbps"
          },
          "price": {
            "title": "Cài đặt giá",
            "base_price": "Giá cơ bản",
            "clear": {
              "button": "Xóa",
              "tooltip": "Xóa tất cả giá"
            }
          },
          "device": {
            "label": "Giới hạn thiết bị",
            "placeholder": "Không giới hạn",
            "unit": "cái"
          },
          "capacity": {
            "label": "Giới hạn dung lượng",
            "placeholder": "Không giới hạn",
            "unit": "người"
          },
          "reset_method": {
            "label": "Cách reset",
            "placeholder": "Chọn cách reset",
            "description": "Cách tính traffic cho gói này.",
            "options": {
              "follow_system": "Theo hệ thống",
              "monthly_first": "Ngày 1 hàng tháng",
              "monthly_reset": "Ngày mua",
              "no_reset": "Không reset",
              "yearly_first": "Ngày 1 tháng 1",
              "yearly_reset": "Hàng năm"
            }
          },
          "content": {
            "label": "Mô tả gói",
            "placeholder": "Nhập mô tả...",
            "description": "Hỗ trợ Markdown. Danh sách, in đậm, v.v.",
            "preview": "Xem trước",
            "preview_button": {
              "show": "Hiện",
              "hide": "Ẩn"
            },
            "template": {
              "button": "Mẫu",
              "tooltip": "Chèn mô tả chuẩn",
              "content": "## Đặc điểm gói\n• Tốc độ cao và ổn định\n• Hỗ trợ nhiều thiết bị\n• Reset traffic định kỳ\n\n## Hướng dẫn\n1. Hỗ trợ: iOS, Android, Windows, macOS\n2. Hỗ trợ kỹ thuật 24/7\n3. Tự động cập nhật traffic\n\n## Lưu ý\n- Cấm lạm dụng\n- Tuân thủ pháp luật\n- Có thể đổi gói bất cứ lúc nào"
            }
          },
          "force_update": {
            "label": "Bắt buộc cập nhật cho tất cả"
          },
          "submit": {
            "submitting": "Đang lưu...",
            "submit": "Áp dụng",
            "cancel": "Hủy",
            "success": {
              "add": "Đã thêm gói",
              "update": "Đã cập nhật gói"
            }
          }
        },
        "page": {
          "description": "Cài đặt gói đăng ký: giới hạn, giá và hiển thị."
        }
      }
    },
    "email": {
      "title": "Cài đặt Email",
      "description": "Cấu hình SMTP để gửi mã xác nhận, khôi phục mật khẩu và thông báo hệ thống.",
      "tab_settings": "Cài đặt",
      "tab_templates": "Mẫu",
      "email_host": {
        "title": "SMTP Server",
        "description": "Địa chỉ server, vd: smtp.gmail.com"
      },
      "email_port": {
        "title": "SMTP Port",
        "description": "Thường là: 465 (SSL) hoặc 587 (TLS)"
      },
      "email_username": {
        "title": "Tên người dùng SMTP",
        "description": "Email hoặc tên đăng nhập của bạn"
      },
      "email_password": {
        "title": "Mật khẩu SMTP",
        "description": "Mật khẩu ứng dụng hoặc mật khẩu chính"
      },
      "email_encryption": {
        "title": "Mã hóa",
        "description": "Phương thức bảo vệ kết nối",
        "none": "Không",
        "ssl": "SSL/TLS",
        "tls": "STARTTLS"
      },
      "email_from": {
        "title": "Email người gửi",
        "description": "Sẽ hiển thị trong tiêu đề email"
      },
      "email_from_name": {
        "title": "Tên người gửi",
        "description": "Vd: 'Quản trị Lets VPN'"
      },
      "email_template": {
        "title": "Mẫu Email",
        "description": "Chọn kiểu trang trí thông báo",
        "placeholder": "Chọn mẫu"
      },
      "remind_mail": {
        "title": "Email nhắc nhở",
        "description": "Thông báo người dùng khi hết traffic hoặc hạn dùng."
      },
      "test": {
        "title": "Email test",
        "sending": "Đang gửi...",
        "description": "Kiểm tra cài đặt bằng cách gửi email cho chính mình",
        "success": "Gửi email thành công",
        "error": "Lỗi khi gửi test"
      }
    },
    "telegram": {
      "title": "Cài đặt Telegram",
      "description": "Kết nối với Bot Telegram: thông báo, liên kết tài khoản và lệnh.",
      "bot_token": {
        "title": "Token Bot",
        "description": "Lấy từ @BotFather.",
        "placeholder": "0000000000:xxxxxxxxx_xxxxxxxxxxxxxxx"
      },
      "webhook_url": {
        "title": "Webhook Base URL",
        "description": "Để trống để dùng URL trang web. Hệ thống sẽ tự thêm đường dẫn cần thiết.",
        "docs": "Tài liệu Telegram Webhook",
        "placeholder": "https://example.com"
      },
      "webhook": {
        "title": "Cài đặt Webhook",
        "description": "Nếu không cài đặt, bot sẽ không nhận được tin nhắn từ người dùng.",
        "button": "Cài đặt 1-click",
        "setting": "Đang cài đặt...",
        "success": "Cài đặt Webhook thành công",
        "target_default": "Dùng URL chính của trang web.",
        "target_custom": "Dùng URL tùy chỉnh: {{url}}",
        "debug": {
          "title": "Debug Webhook",
          "success": "Thành công",
          "url": "URL đầy đủ",
          "baseUrl": "Base URL"
        }
      },
      "bot_enable": {
        "title": "Hướng dẫn liên kết",
        "description": "Hiển thị hướng dẫn liên kết Telegram ở phía người dùng."
      },
      "discuss_link": {
        "title": "Link nhóm/chat",
        "description": "Sẽ hiển thị trong trang cá nhân.",
        "placeholder": "https://t.me/your_chat"
      }
    },
    "app": {
      "title": "Cài đặt Ứng dụng",
      "description": "Cấu hình liên kết tải xuống và phiên bản client cho các nền tảng khác nhau.",
      "common": {
        "placeholder": "Nhập giá trị"
      },
      "windows": {
        "version": {
          "title": "Phiên bản Windows",
          "description": "Phiên bản client Windows hiện tại"
        },
        "download": {
          "title": "Link tải Windows",
          "description": "URL tải file .exe"
        }
      },
      "macos": {
        "version": {
          "title": "Phiên bản macOS",
          "description": "Phiên bản client macOS hiện tại"
        },
        "download": {
          "title": "Link tải macOS",
          "description": "URL tải file .dmg"
        }
      },
      "android": {
        "version": {
          "title": "Phiên bản Android",
          "description": "Phiên bản .apk hiện tại"
        },
        "download": {
          "title": "Link tải Android",
          "description": "URL tải APK"
        }
      }
    },
    "common": {
      "saving": "Đang lưu...",
      "save_success": "Đã tự động lưu",
      "placeholder": "Văn bản...",
      "autoSaved": "Đã lưu"
    },
    "subscribe_template": {
      "title": "Mẫu Đăng ký",
      "description": "Cấu hình định dạng đầu ra cho các client khác nhau.",
      "singbox": {
        "title": "Mẫu Sing-box",
        "description": "Định dạng cấu hình cho Sing-box"
      },
      "clash": {
        "title": "Mẫu Clash",
        "description": "Định dạng cấu hình cho Clash"
      },
      "clashmeta": {
        "title": "Mẫu Clash Meta",
        "description": "Định dạng cấu hình cho Clash Meta (Mihomo)"
      },
      "stash": {
        "title": "Mẫu Stash",
        "description": "Định dạng cấu hình cho Stash (iOS)"
      },
      "surge": {
        "title": "Mẫu Surge",
        "description": "Định dạng cấu hình cho Surge"
      },
      "surfboard": {
        "title": "Mẫu Surfboard",
        "description": "Định dạng cấu hình cho Surfboard"
      }
    },
    "email_template": {
      "title": "Mẫu Email",
      "description": "Cấu hình mẫu thông báo email hệ thống",
      "customized": "Đã sửa",
      "subject": "Chủ đề",
      "subject_placeholder": "Nhập chủ đề email, hỗ trợ {{name}} và các biến khác",
      "content": "Nội dung mẫu (HTML)",
      "preview": "Xem trước",
      "override_hint": "Lưu sẽ ghi đè mẫu hệ thống mặc định. Nhấn «Mặc định» để khôi phục mẫu giao diện.",
      "placeholders": "Biến khả dụng",
      "var_name": "Biến",
      "var_desc": "Mô tả",
      "var_sample": "Ví dụ",
      "required": "Bắt buộc",
      "insert": "Chèn",
      "placeholder_hint": "* biến bắt buộc. Nhấn để chèn vào cuối nội dung.",
      "click_to_insert": "Nhấn để chèn",
      "save": "Lưu",
      "save_success": "Đã lưu mẫu",
      "save_before_test": "Lưu thay đổi trước khi gửi email test",
      "send_test": "Email test",
      "test_dialog_title": "Gửi email test",
      "test_dialog_description": "Nhập email người nhận. Để trống để gửi đến email admin của bạn.",
      "test_email_placeholder": "Email người nhận (trống = tài khoản hiện tại)",
      "sending": "Đang gửi...",
      "test_success": "Đã gửi email test",
      "reset": "Mặc định",
      "reset_title": "Đặt lại mẫu",
      "reset_description": "Bạn có chắc chắn muốn đặt lại mẫu? Thay đổi của bạn sẽ bị mất.",
      "reset_confirm": "Xác nhận",
      "reset_success": "Đã đặt lại mẫu",
      "unsaved": "Có thay đổi chưa lưu",
      "discard_title": "Thay đổi chưa lưu",
      "discard_description": "Mẫu này có thay đổi chưa lưu. Chuyển tab sẽ làm mất chúng.",
      "discard_confirm": "Hủy thay đổi",
      "cancel": "Hủy"
    }
  },
  "sidebar": {
    "dashboard": "Bảng điều khiển",
    "systemManagement": "Quản lý Hệ thống",
    "systemConfig": "Cấu hình Hệ thống",
    "themeConfig": "Cấu hình Giao diện",
    "noticeManagement": "Thông báo",
    "paymentConfig": "Phương thức Thanh toán",
    "knowledgeManagement": "Kho Kiến thức",
    "nodeManagement": "Node",
    "permissionGroupManagement": "Nhóm Quyền",
    "routeManagement": "Route",
    "subscriptionManagement": "Quản lý Đăng ký",
    "planManagement": "Gói",
    "orderManagement": "Đơn hàng",
    "couponManagement": "Mã Giảm Giá",
    "userManagement": "Người Dùng",
    "ticketManagement": "Ticket"
  },
  "ticket": {
    "title": "Quản lý Ticket",
    "description": "Xem và quản lý yêu cầu của người dùng, bao gồm trả lời và đóng ticket.",
    "columns": {
      "id": "ID Ticket",
      "subject": "Chủ đề",
      "level": "Ưu tiên",
      "status": "Trạng thái",
      "updated_at": "Cập nhật",
      "created_at": "Tạo",
      "actions": "Thao tác"
    },
    "status": {
      "closed": "Đã đóng",
      "replied": "Đã trả lời",
      "pending": "Chờ xử lý",
      "processing": "Đang xử lý"
    },
    "level": {
      "low": "Thấp",
      "medium": "Trung bình",
      "high": "Cao"
    },
    "filter": {
      "placeholder": "Tìm {field}...",
      "no_results": "Không tìm thấy kết quả",
      "selected": "Đã chọn: {count}",
      "clear": "Xóa bộ lọc"
    },
    "actions": {
      "view_details": "Chi tiết",
      "close_ticket": "Đóng Ticket",
      "close_confirm_title": "Xác nhận đóng",
      "close_confirm_description": "Bạn có chắc chắn muốn đóng ticket này? Nó sẽ chuyển vào danh sách đã đóng, nhưng vẫn có thể trả lời.",
      "close_confirm_button": "Đóng Ticket",
      "close_success": "Đóng ticket thành công",
      "view_ticket": "Xem Ticket"
    },
    "detail": {
      "no_messages": "Không có tin nhắn",
      "created_at": "Đã tạo",
      "sender_admin": "Admin",
      "sender_user": "Người dùng",
      "user_info": "Thông tin người dùng",
      "traffic_records": "Lịch sử Traffic",
      "order_records": "Lịch sử Đơn hàng",
      "input": {
        "closed_reply_placeholder": "Ticket đã đóng, nhưng bạn vẫn có thể trả lời...",
        "closed_hint": "Ticket đã đóng, nhưng bạn vẫn có thể trả lời. Tin nhắn mới sẽ được thêm vào ticket này.",
        "reply_placeholder": "Nhập câu trả lời của bạn...",
        "sending": "Đang gửi...",
        "send": "Gửi"
      }
    },
    "list": {
      "title": "Danh sách Ticket",
      "search_placeholder": "Tìm theo chủ đề hoặc email",
      "no_tickets": "Không có Ticket",
      "no_open_tickets": "Không có Ticket mở",
      "no_closed_tickets": "Không có Ticket đóng",
      "no_search_results": "Không tìm thấy Ticket",
      "collapse": "Thu gọn danh sách",
      "expand": "Mở rộng danh sách"
    }
  },
  "payment": {
    "title": "Phương thức Thanh toán",
    "description": "Cấu hình các phương thức thanh toán, bao gồm Alipay, WeChat Pay, v.v.",
    "table": {
      "columns": {
        "id": "ID",
        "enable": "Bật",
        "name": "Tên",
        "payment": "Cổng thanh toán",
        "notify_url": "URL Thông báo",
        "notify_url_tooltip": "Cổng thanh toán sẽ gửi thông báo đến địa chỉ này. Đảm bảo nó có thể truy cập qua firewall của bạn.",
        "actions": "Thao tác"
      },
      "actions": {
        "edit": "Sửa",
        "delete": {
          "title": "Xác nhận xóa",
          "description": "Bạn có chắc chắn muốn xóa phương thức thanh toán này? Hành động này không thể hoàn tác.",
          "success": "Xóa thành công"
        }
      },
      "toolbar": {
        "search": "Tìm phương thức thanh toán...",
        "reset": "Đặt lại",
        "sort": {
          "hint": "Kéo thả để sắp xếp, sau đó nhấn lưu",
          "save": "Lưu thứ tự",
          "edit": "Sửa thứ tự"
        }
      }
    },
    "form": {
      "add": {
        "button": "Thêm phương thức thanh toán",
        "title": "Thêm phương thức thanh toán"
      },
      "edit": {
        "title": "Sửa phương thức thanh toán"
      },
      "fields": {
        "name": {
          "label": "Tên hiển thị",
          "placeholder": "Nhập tên",
          "description": "Dùng để hiển thị cho người dùng"
        },
        "icon": {
          "label": "URL Icon",
          "placeholder": "https://example.com/icon.svg",
          "description": "URL icon để hiển thị"
        },
        "notify_domain": {
          "label": "Domain Thông báo",
          "placeholder": "https://example.com",
          "description": "Domain nhận thông báo từ cổng thanh toán"
        },
        "handling_fee_percent": {
          "label": "Phí (%)",
          "placeholder": "0-100"
        },
        "handling_fee_fixed": {
          "label": "Phí cố định",
          "placeholder": "0"
        },
        "payment": {
          "label": "Cổng Thanh toán",
          "placeholder": "Chọn cổng",
          "description": "Chọn cổng kỹ thuật để xử lý thanh toán"
        }
      },
      "validation": {
        "name": {
          "min": "Tên phải có ít nhất 2 ký tự",
          "max": "Tên không được vượt quá 30 ký tự"
        },
        "notify_domain": {
          "url": "Vui lòng nhập URL hợp lệ"
        },
        "payment": {
          "required": "Vui lòng chọn cổng thanh toán"
        }
      },
      "buttons": {
        "cancel": "Hủy",
        "submit": "Gửi"
      },
      "sections": {
        "payment_config": "Cấu hình Cổng"
      },
      "messages": {
        "success": "Lưu thành công"
      }
    }
  },
  "subscribe": {
    "plan": {
      "title": "Gói Đăng ký",
      "add": "Thêm Gói",
      "search": "Tìm gói...",
      "sort": {
        "edit": "Sửa thứ tự",
        "save": "Lưu thứ tự"
      },
      "columns": {
        "id": "ID",
        "show": "Hiển thị",
        "sell": "Bán",
        "renew": "Gia hạn",
        "renew_tooltip": "Người dùng hiện tại có thể gia hạn khi gói ngừng bán không",
        "name": "Tên",
        "stats": "Thống kê",
        "group": "Nhóm Truy cập",
        "price": "Giá",
        "actions": "Thao tác",
        "edit": "Sửa",
        "delete": "Xóa",
        "delete_confirm": {
          "title": "Xác nhận xóa",
          "description": "Hành động này sẽ xóa vĩnh viễn gói đăng ký này. Bạn có chắc chắn muốn tiếp tục?",
          "success": "Xóa gói thành công"
        },
        "price_period": {
          "monthly": "1 Tháng",
          "quarterly": "3 Tháng",
          "half_yearly": "6 Tháng",
          "yearly": "1 năm",
          "two_yearly": "2 năm",
          "three_yearly": "3 năm",
          "onetime": "Vĩnh viễn",
          "reset_traffic": "Đặt lại dung lượng",
          "no_price": "Không có giá",
          "unit": {
            "month": "/tháng",
            "quarter": "/quý",
            "half_year": "/6tháng",
            "year": "/năm",
            "two_year": "/2năm",
            "three_year": "/3năm",
            "times": "/lần"
          }
        }
      },
      "form": {
        "add_title": "Thêm Gói",
        "edit_title": "Sửa Gói",
        "name": {
          "label": "Tên Gói",
          "placeholder": "Nhập tên gói"
        },
        "group": {
          "label": "Nhóm Server",
          "add": "Thêm Nhóm",
          "placeholder": "Chọn nhóm server"
        },
        "transfer": {
          "label": "Traffic",
          "placeholder": "Nhập giới hạn traffic",
          "unit": "GB"
        },
        "speed": {
          "label": "Giới hạn Tốc độ",
          "placeholder": "Nhập giới hạn tốc độ",
          "unit": "Mbps"
        },
        "price": {
          "title": "Cài đặt Giá",
          "base_price": "Giá cơ bản",
          "clear": {
            "button": "Xóa",
            "tooltip": "Xóa tất cả giá"
          },
          "period": {
            "monthly": "1 Tháng",
            "months": "{{count}} Tháng"
          },
          "onetime_desc": "Gói traffic một lần, không giới hạn thời gian",
          "reset_desc": "Gói reset traffic, có thể dùng nhiều lần"
        },
        "device": {
          "label": "Giới hạn Thiết bị",
          "placeholder": "Nhập giới hạn thiết bị",
          "unit": "Thiết bị"
        },
        "capacity": {
          "label": "Giới hạn Người dùng",
          "placeholder": "Nhập giới hạn dung lượng",
          "unit": "Người dùng"
        },
        "tags": {
          "label": "Thẻ",
          "placeholder": "Nhập thẻ và nhấn Enter"
        },
        "reset_method": {
          "label": "Phương pháp Reset Traffic",
          "placeholder": "Chọn phương pháp reset",
          "description": "Phương pháp reset xác định khi nào và như thế nào traffic người dùng được reset",
          "options": {
            "follow_system": "Theo cài đặt hệ thống",
            "monthly_first": "Ngày 1 hàng tháng",
            "monthly_reset": "Hàng tháng (ngày mua)",
            "no_reset": "Không reset",
            "yearly_first": "Ngày 1 đầu năm",
            "yearly_reset": "Hàng năm (ngày mua)"
          }
        },
        "content": {
          "label": "Mô tả Gói",
          "placeholder": "Nhập mô tả gói",
          "description": "Hỗ trợ định dạng Markdown",
          "preview": "Xem trước",
          "preview_button": {
            "show": "Hiện xem trước",
            "hide": "Ẩn xem trước"
          },
          "template": {
            "button": "Dùng Mẫu",
            "tooltip": "Dùng mẫu chuẩn",
            "content": "## Chi tiết Gói\n\n- Traffic: {{transfer}} GB\n- Tốc độ: {{speed}} Mbps\n- Thiết bị: {{devices}}\n\n## Thông tin Dịch vụ\n\n1. Reset Traffic: {{reset_method}}\n2. Hỗ trợ mọi nền tảng\n3. Hỗ trợ kỹ thuật 24/7"
          }
        },
        "force_update": {
          "label": "Bắt buộc cập nhật gói người dùng"
        },
        "submit": {
          "cancel": "Hủy",
          "submit": "Gửi",
          "submitting": "Đang gửi...",
          "success": {
            "add": "Thêm gói thành công",
            "update": "Cập nhật gói thành công"
          },
          "error": {
            "validation": "Lỗi xác thực biểu mẫu. Vui lòng kiểm tra dữ liệu."
          }
        }
      },
      "page": {
        "description": "Tại đây bạn có thể cấu hình các gói đăng ký, bao gồm thêm, xóa và sửa."
      }
    }
  },
  "user": {
    "manage": {
      "title": "Quản lý Người dùng",
      "description": "Tại đây bạn có thể quản lý người dùng, bao gồm thêm, xóa, sửa và tìm kiếm."
    },
    "columns": {
      "is_admin": "Admin",
      "is_staff": "Nhân viên",
      "id": "ID",
      "email": "Email",
      "online_count": "Thiết bị Online",
      "status": "Trạng thái",
      "subscription": "Đăng ký",
      "group": "Nhóm",
      "used_traffic": "Đã dùng",
      "total_traffic": "Tổng Traffic",
      "expire_time": "Hạn dùng",
      "balance": "Số dư",
      "commission": "Hoa hồng",
      "register_time": "Ngày đăng ký",
      "actions": "Thao tác",
      "next_reset_at": "Reset tiếp theo",
      "device_limit": {
        "unlimited": "Không giới hạn thiết bị",
        "limited": "Tối đa {{count}} thiết bị"
      },
      "status_text": {
        "normal": "Bình thường",
        "banned": "Đã cấm"
      },
      "online_status": {
        "online": "Đang Online",
        "never": "Chưa bao giờ Online",
        "last_online": "Online cuối: {{time}}",
        "offline_duration": {
          "days": "Offline: {{count}}ng",
          "hours": "Offline: {{count}}g",
          "minutes": "Offline: {{count}}p",
          "seconds": "Offline: {{count}}s"
        }
      },
      "expire_status": {
        "permanent": "Vĩnh viễn",
        "expired": "Hết hạn {{days}} ngày trước",
        "remaining": "Còn {{days}} ngày"
      },
      "actions_menu": {
        "edit": "Sửa",
        "view_details": "Xem chi tiết",
        "assign_order": "Gán Đơn hàng",
        "copy_url": "Sao chép Link Đăng ký",
        "reset_secret": "Reset UUID và URL",
        "orders": "Đơn hàng",
        "invites": "Lời mời",
        "traffic_records": "Lịch sử Traffic",
        "reset_traffic": "Đặt lại dung lượng",
        "delete": "Xóa",
        "delete_confirm_title": "Xác nhận xóa Người dùng",
        "delete_confirm_description": "Hành động này sẽ xóa vĩnh viễn người dùng {{email}} và tất cả dữ liệu liên quan, bao gồm đơn hàng, mã giảm giá, lịch sử traffic và ticket. Không thể hoàn tác. Tiếp tục?"
      }
    },
    "filter": {
      "selected": "Đã chọn {{count}}",
      "no_results": "Không tìm thấy kết quả.",
      "clear": "Xóa bộ lọc",
      "search_placeholder": "Tìm kiếm...",
      "email_search": "Tìm theo email...",
      "advanced": "Bộ lọc Nâng cao",
      "reset": "Đặt lại bộ lọc",
      "sheet": {
        "title": "Bộ lọc Nâng cao",
        "description": "Thêm một hoặc nhiều điều kiện để tìm kiếm người dùng chính xác",
        "conditions": "Điều kiện lọc",
        "add": "Thêm điều kiện",
        "condition": "Điều kiện {{number}}",
        "field": "Chọn trường",
        "operator": "Chọn toán tử",
        "value": "Nhập giá trị",
        "value_number": "Nhập giá trị ({{unit}})",
        "reset": "Đặt lại",
        "apply": "Áp dụng bộ lọc"
      },
      "fields": {
        "email": "Email",
        "id": "ID Người dùng",
        "plan_id": "Đăng ký",
        "transfer_enable": "Traffic",
        "total_used": "Đã dùng",
        "online_count": "Thiết bị Online",
        "expired_at": "Hạn dùng",
        "uuid": "UUID",
        "token": "Token",
        "banned": "Trạng thái Tài khoản",
        "remark": "Ghi chú",
        "inviter_email": "Email Người mời",
        "invite_user_id": "ID Người mời",
        "is_admin": "Admin",
        "is_staff": "Nhân viên"
      },
      "operators": {
        "contains": "Chứa",
        "eq": "Bằng",
        "gt": "Lớn hơn",
        "lt": "Nhỏ hơn"
      },
      "status": {
        "normal": "Bình thường",
        "banned": "Đã cấm"
      },
      "boolean": {
        "true": "Có",
        "false": "Không"
      }
    },
    "generate": {
      "button": "Tạo Người dùng",
      "title": "Tạo Người dùng",
      "form": {
        "email": "Email",
        "email_prefix": "Tiền tố tài khoản (để trống để tạo hàng loạt)",
        "email_domain": "Domain",
        "password": "Mật khẩu",
        "password_placeholder": "Để trống để dùng email làm mật khẩu",
        "expire_time": "Hạn dùng",
        "expire_time_placeholder": "Chọn hạn dùng, để trống cho vĩnh viễn",
        "permanent": "Vĩnh viễn",
        "subscription": "Gói Đăng ký",
        "subscription_none": "Không có",
        "generate_count": "Số lượng tạo",
        "generate_count_placeholder": "Nhập số lượng để tạo hàng loạt",
        "cancel": "Hủy",
        "submit": "Tạo",
        "success": "Đã tạo người dùng",
        "download_csv": "Tải CSV"
      }
    },
    "edit": {
      "button": "Sửa Người dùng",
      "title": "Quản lý Người dùng",
      "form": {
        "email": "Email",
        "email_placeholder": "Vui lòng nhập email",
        "inviter_email": "Email Người mời",
        "inviter_email_placeholder": "Vui lòng nhập email",
        "password": "Mật khẩu",
        "password_placeholder": "Nhập mật khẩu mới nếu muốn đổi",
        "balance": "Số dư",
        "balance_placeholder": "Vui lòng nhập số dư",
        "commission_balance": "Số dư Hoa hồng",
        "commission_balance_placeholder": "Vui lòng nhập số dư hoa hồng",
        "upload": "Traffic (Upload)",
        "upload_placeholder": "Nhập traffic upload",
        "download": "Traffic (Download)",
        "download_placeholder": "Nhập traffic download",
        "total_traffic": "Tổng Traffic",
        "total_traffic_placeholder": "Vui lòng nhập traffic",
        "expire_time": "Hạn dùng",
        "expire_time_placeholder": "Chọn ngày, để trống cho vĩnh viễn",
        "expire_time_specific": "Thời gian cụ thể",
        "expire_time_today": "Đến hết hôm nay",
        "expire_time_permanent": "Vĩnh viễn",
        "expire_time_1month": "Một tháng",
        "expire_time_3months": "Ba tháng",
        "expire_time_confirm": "Xác nhận",
        "subscription": "Gói Đăng ký",
        "subscription_none": "Không có",
        "account_status": "Trạng thái Tài khoản",
        "commission_type": "Loại Hoa hồng",
        "commission_type_system": "Theo hệ thống",
        "commission_type_cycle": "Hoa hồng chu kỳ",
        "commission_type_onetime": "Hoa hồng một lần",
        "commission_rate": "Tỷ lệ Hoa hồng",
        "commission_rate_placeholder": "Để trống cho tỷ lệ hệ thống",
        "discount": "Giảm giá Độc quyền",
        "discount_placeholder": "Để trống nếu không có giảm giá",
        "speed_limit": "Giới hạn Tốc độ",
        "speed_limit_placeholder": "Để trống cho không giới hạn",
        "device_limit": "Giới hạn Thiết bị",
        "device_limit_placeholder": "Để trống cho không giới hạn",
        "is_admin": "Admin",
        "is_staff": "Nhân viên",
        "remarks": "Ghi chú",
        "remarks_placeholder": "Nhập ghi chú tại đây",
        "cancel": "Hủy",
        "submit": "Gửi",
        "success": "Sửa thành công"
      }
    },
    "actions": {
      "title": "Thao tác",
      "send_email": "Gửi Email",
      "export_csv": "Xuất CSV",
      "traffic_reset_stats": "Thống kê Reset Traffic",
      "batch_ban": "Cấm Hàng loạt",
      "confirm_ban": {
        "title": "Xác nhận Cấm Hàng loạt",
        "filtered_description": "Hành động này sẽ cấm tất cả người dùng khớp với bộ lọc hiện tại. Không thể hoàn tác.",
        "all_description": "Hành động này sẽ cấm tất cả người dùng trong hệ thống. Không thể hoàn tác.",
        "cancel": "Hủy",
        "confirm": "Xác nhận Cấm",
        "banning": "Đang cấm..."
      }
    },
    "messages": {
      "success": "Thành công",
      "error": "Lỗi",
      "export": {
        "success": "Xuất thành công",
        "failed": "Lỗi xuất"
      },
      "batch_ban": {
        "success": "Cấm hàng loạt thành công",
        "failed": "Lỗi cấm hàng loạt"
      },
      "send_mail": {
        "success": "Đã gửi Email",
        "failed": "Lỗi gửi email",
        "required_fields": "Vui lòng điền các trường bắt buộc"
      }
    },
    "traffic_reset": {
      "title": "Reset Traffic",
      "description": "Reset traffic cho người dùng {{email}}",
      "tabs": {
        "reset": "Reset Traffic",
        "history": "Lịch sử Reset"
      },
      "user_info": "Thông tin Người dùng",
      "warning": {
        "title": "Thông báo Quan trọng",
        "irreversible": "Thao tác reset traffic không thể đảo ngược, vui lòng cẩn thận",
        "reset_to_zero": "Sau khi reset, traffic upload và download của người dùng sẽ về 0",
        "logged": "Mọi thao tác reset đều được ghi vào nhật ký hệ thống"
      },
      "reason": {
        "label": "Lý do Reset",
        "placeholder": "Nhập lý do reset (tùy chọn)",
        "optional": "Trường này không bắt buộc, dùng để ghi lại lý do reset"
      },
      "confirm_reset": "Xác nhận Reset",
      "resetting": "Đang reset...",
      "reset_success": "Reset traffic thành công",
      "reset_failed": "Lỗi khi reset traffic",
      "history": {
        "summary": "Tổng quan Reset",
        "reset_count": "Số lần Reset",
        "last_reset": "Reset cuối",
        "next_reset": "Reset tiếp theo",
        "never": "Chưa bao giờ reset",
        "no_schedule": "Không lên lịch reset",
        "records": "Bản ghi Reset",
        "recent_records": "10 bản ghi gần nhất",
        "no_records": "Không có bản ghi reset",
        "reset_time": "Thời gian Reset",
        "traffic_cleared": "Traffic đã xóa"
      },
      "stats": {
        "title": "Thống kê Reset Traffic",
        "description": "Xem thống kê hệ thống về reset traffic",
        "time_range": "Khoảng thời gian",
        "total_resets": "Tổng số Reset",
        "auto_resets": "Tự động",
        "manual_resets": "Thủ công",
        "cron_resets": "Theo lịch/Cron",
        "in_period": "Trong {{days}} ngày qua",
        "breakdown": "Phân loại",
        "breakdown_description": "Tỷ lệ phần trăm các loại thao tác reset",
        "auto_percentage": "Tỷ lệ Auto Reset",
        "manual_percentage": "Tỷ lệ Manual Reset",
        "cron_percentage": "Tỷ lệ Cron Reset",
        "days_options": {
          "week": "1 Tuần",
          "month": "1 Tháng",
          "quarter": "1 Quý",
          "year": "1 Năm"
        }
      }
    },
    "traffic_reset_logs": {
      "title": "Nhật ký Reset Traffic",
      "description": "Xem chi tiết tất cả các thao tác reset traffic trong hệ thống",
      "columns": {
        "id": "ID Nhật ký",
        "user": "Người dùng",
        "reset_type": "Loại Reset",
        "trigger_source": "Nguồn kích hoạt",
        "cleared_traffic": "Traffic đã xóa",
        "cleared": "Đã xóa",
        "upload": "Upload",
        "download": "Download",
        "reset_time": "Thời gian Reset",
        "log_time": "Thời gian Ghi"
      },
      "filters": {
        "search_user": "Tìm theo email...",
        "reset_type": "Loại Reset",
        "trigger_source": "Nguồn",
        "all_types": "Tất cả loại",
        "all_sources": "Tất cả nguồn",
        "start_date": "Ngày bắt đầu",
        "end_date": "Ngày kết thúc",
        "apply_date": "Áp dụng",
        "reset": "Đặt lại",
        "filter_title": "Tham số Lọc",
        "filter_description": "Đặt điều kiện để tìm bản ghi reset traffic",
        "reset_types": {
          "monthly": "1 Tháng",
          "first_day_month": "Ngày 1 hàng tháng",
          "yearly": "1 năm",
          "first_day_year": "Ngày 1 đầu năm",
          "manual": "Thủ công"
        },
        "trigger_sources": {
          "auto": "Tự động",
          "manual": "Thủ công",
          "cron": "Theo lịch/Cron"
        }
      },
      "actions": {
        "export": "Xuất Nhật ký",
        "exporting": "Đang xuất...",
        "export_success": "Xuất thành công",
        "export_failed": "Lỗi xuất"
      },
      "trigger_descriptions": {
        "manual": "Reset traffic do admin thực hiện thủ công",
        "cron": "Reset tự động bởi tác vụ hệ thống (Cron)",
        "auto": "Tự động reset khi thỏa mãn điều kiện hệ thống",
        "other": "Reset bằng phương thức khác"
      }
    },
    "send_mail": {
      "title": "Gửi Email",
      "description": "Gửi tin nhắn đến người dùng đã chọn hoặc được lọc",
      "subject": "Chủ đề",
      "content": "Nội dung",
      "sending": "Đang gửi...",
      "send": "Gửi"
    },
    "dialog": {
      "title": "Chi tiết Người dùng",
      "basicInfo": "Thông tin Cơ bản",
      "subscriptionInfo": "Thông tin Đăng ký",
      "trafficInfo": "Thông tin Traffic",
      "financialInfo": "Thông tin Tài chính",
      "activityInfo": "Hoạt động",
      "inviteInfo": "Thông tin Giới thiệu",
      "timeInfo": "Mốc Thời gian",
      "subscriptionUrl": "Link Đăng ký",
      "fields": {
        "userId": "ID Người dùng",
        "email": "Email",
        "uuid": "UUID",
        "token": "Token",
        "remarks": "Ghi chú",
        "subscriptionPlan": "Gói",
        "permissionGroup": "Nhóm Truy cập",
        "expiredAt": "Hết hạn",
        "deviceLimit": "Giới hạn Thiết bị",
        "speedLimit": "Giới hạn Tốc độ",
        "transferEnable": "Tổng Traffic",
        "uploadUsed": "Đã dùng (Up)",
        "downloadUsed": "Đã dùng (Down)",
        "totalUsed": "Tổng đã dùng",
        "lastResetAt": "Reset cuối",
        "nextResetAt": "Reset tiếp theo",
        "resetCount": "Số lần Reset",
        "balance": "Số dư",
        "commissionBalance": "Số dư Hoa hồng",
        "commissionType": "Loại Hoa hồng",
        "commissionRate": "Tỷ lệ Hoa hồng",
        "lastLoginAt": "Đăng nhập cuối",
        "lastLoginIp": "IP Đăng nhập cuối",
        "lastOnlineAt": "Online cuối",
        "onlineCount": "Thiết bị Online",
        "inviteUser": "Người mời",
        "inviteUserId": "ID Người mời",
        "createdAt": "Ngày Tạo",
        "updatedAt": "Ngày Cập nhật",
        "subscribeUrl": "Link Đăng ký",
        "telegramId": "Telegram ID"
      }
    },
    "status": {
      "normal": "Bình thường",
      "banned": "Đã cấm",
      "admin": "Admin",
      "staff": "Nhân viên"
    }
  },
  "giftCard": {
    "common": {
      "search": "Tìm kiếm...",
      "enabled": "Hoạt động",
      "disabled": "Vô hiệu hoá"
    },
    "types": {
      "1": "Thẻ chung",
      "2": "Thẻ gói",
      "3": "Thẻ bí ẩn",
      "4": "Loại 4",
      "5": "Loại 5",
      "6": "Loại 6",
      "7": "Loại 7",
      "8": "Loại 8",
      "9": "Loại 9",
      "10": "Loại 10"
    },
    "title": "Quản lý Thẻ Quà Tặng",
    "description": "Quản lý mẫu thẻ quà tặng, mã kích hoạt và lịch sử sử dụng.",
    "tabs": {
      "templates": "Quản lý Mẫu",
      "codes": "Mã Kích hoạt",
      "usages": "Lịch sử Sử dụng",
      "statistics": "Thống kê"
    },
    "template": {
      "title": "Quản lý Mẫu",
      "description": "Tạo, sửa và xóa mẫu thẻ quà tặng.",
      "table": {
        "title": "Danh sách Mẫu",
        "columns": {
          "id": "ID",
          "name": "Tên",
          "type": "Loại",
          "status": "Trạng thái",
          "sort": "Sắp xếp",
          "rewards": "Phần thưởng",
          "created_at": "Ngày tạo",
          "actions": "Thao tác",
          "no_rewards": "Không có phần thưởng"
        }
      },
      "form": {
        "add": "Thêm Mẫu",
        "edit": "Sửa Mẫu",
        "name": {
          "label": "Tên Mẫu",
          "placeholder": "Nhập tên mẫu",
          "required": "Vui lòng nhập tên mẫu"
        },
        "sort": {
          "label": "Sắp xếp",
          "placeholder": "Số nhỏ hiển thị trước"
        },
        "type": {
          "label": "Loại",
          "placeholder": "Chọn loại thẻ"
        },
        "description": {
          "label": "Mô tả",
          "placeholder": "Nhập mô tả thẻ quà tặng"
        },
        "status": {
          "label": "Trạng thái",
          "description": "Nếu tắt, mẫu này không thể dùng để tạo hoặc kích hoạt thẻ mới."
        },
        "display": {
          "title": "Hiệu ứng Hình ảnh"
        },
        "theme_color": {
          "label": "Màu Chủ đề"
        },
        "icon": {
          "label": "Icon",
          "placeholder": "Nhập URL Icon"
        },
        "background_image": {
          "label": "Ảnh Nền",
          "placeholder": "Nhập URL Ảnh Nền"
        },
        "conditions": {
          "title": "Điều kiện Sử dụng",
          "new_user_max_days": {
            "label": "Giới hạn ngày đăng ký người dùng mới",
            "placeholder": "Ví dụ: 7 (Chỉ cho người dùng đăng ký trong vòng 7 ngày)"
          },
          "new_user_only": {
            "label": "Chỉ dành cho Người dùng Mới"
          },
          "paid_user_only": {
            "label": "Chỉ dành cho Người dùng Trả phí"
          },
          "require_invite": {
            "label": "Yêu cầu có Người mời"
          },
          "allowed_plans": {
            "label": "Gói Cho phép",
            "placeholder": "Chọn gói có thể kích hoạt (để trống cho tất cả)"
          },
          "disallowed_plans": {
            "label": "Gói Cấm",
            "placeholder": "Chọn gói không được phép kích hoạt"
          }
        },
        "limits": {
          "title": "Giới hạn",
          "max_use_per_user": {
            "label": "Số lần dùng tối đa mỗi người",
            "placeholder": "Để trống cho không giới hạn"
          },
          "cooldown_hours": {
            "label": "Thời gian chờ (giờ) cho thẻ cùng loại",
            "placeholder": "Để trống cho không giới hạn"
          },
          "invite_reward_rate": {
            "label": "Tỷ lệ thưởng cho Người mời",
            "placeholder": "Ví dụ: 0.2 (nghĩa là 20%)",
            "description": "Nếu người dùng có người mời, bonus của họ = bonus người dùng × tỷ lệ này"
          }
        },
        "rewards": {
          "title": "Phần thưởng",
          "balance": {
            "label": "Bonus Số dư (Tiền tệ)",
            "short_label": "Số dư",
            "placeholder": "Nhập số tiền thưởng"
          },
          "transfer_enable": {
            "label": "Bonus Traffic (GB)",
            "short_label": "Traffic",
            "placeholder": "Nhập dung lượng traffic (GB)"
          },
          "expire_days": {
            "label": "Gia hạn (ngày)",
            "short_label": "Hạn",
            "placeholder": "Nhập số ngày"
          },
          "transfer": {
            "label": "Bonus Traffic (Bytes)",
            "placeholder": "Nhập dung lượng traffic (bytes)"
          },
          "days": {
            "label": "Gia hạn (ngày)",
            "placeholder": "Nhập số ngày"
          },
          "device_limit": {
            "label": "Tăng giới hạn Thiết bị",
            "short_label": "Thiết bị",
            "placeholder": "Nhập số thiết bị thêm"
          },
          "reset_package": {
            "label": "Reset Traffic Tháng",
            "description": "Nếu bật, kích hoạt sẽ xóa traffic đã dùng của gói hiện tại người dùng."
          },
          "reset_count": {
            "description": "Loại thẻ này reset việc sử dụng traffic tháng của người dùng."
          },
          "task_card": {
            "description": "Phần thưởng cho thẻ nhiệm vụ được cấu hình trong hệ thống nhiệm vụ."
          },
          "plan_id": {
            "label": "Gói Chỉ định",
            "short_label": "Gói",
            "placeholder": "Vui lòng chọn gói"
          },
          "plan_validity_days": {
            "label": "Thời hạn Gói (ngày)",
            "short_label": "Hạn Gói",
            "placeholder": "Để trống cho hạn mặc định"
          },
          "random_rewards": {
            "label": "Pool Phần thưởng Ngẫu nhiên",
            "add": "Thêm phần thưởng ngẫu nhiên",
            "weight": "Trọng số"
          }
        },
        "special_config": {
          "title": "Cấu hình Đặc biệt",
          "start_time": {
            "label": "Thời gian Bắt đầu Sự kiện",
            "placeholder": "Chọn ngày bắt đầu"
          },
          "end_time": {
            "label": "Thời gian Kết thúc Sự kiện",
            "placeholder": "Chọn ngày kết thúc"
          },
          "festival_bonus": {
            "label": "Hệ số Thưởng Lễ hội",
            "placeholder": "Ví dụ: 1.5 (nghĩa là 1.5x)"
          }
        },
        "submit": {
          "saving": "Đang lưu...",
          "save": "Lưu"
        }
      },
      "actions": {
        "edit": "Sửa",
        "delete": "Xóa",
        "deleteConfirm": {
          "title": "Xác nhận Xóa",
          "description": "Hành động này sẽ xóa vĩnh viễn mẫu này. Bạn có chắc chắn muốn tiếp tục?",
          "confirmText": "Xóa"
        }
      }
    },
    "code": {
      "title": "Quản lý Mã Kích hoạt",
      "form": {
        "generate": "Tạo Mã Kích hoạt",
        "template_id": {
          "label": "Chọn Mẫu",
          "placeholder": "Chọn mẫu để tạo mã kích hoạt"
        },
        "count": {
          "label": "Số lượng Mã"
        },
        "prefix": {
          "label": "Tiền tố Tùy chỉnh (tùy chọn)"
        },
        "expires_hours": {
          "label": "Thời hạn (giờ)"
        },
        "max_usage": {
          "label": "Giới hạn Sử dụng Mã"
        },
        "download_csv": "Xuất CSV",
        "submit": {
          "generating": "Đang tạo...",
          "generate": "Tạo Ngay"
        }
      },
      "description": "Quản lý mã kích hoạt: tạo, xem và xuất.",
      "table": {
        "title": "Danh sách Mã Kích hoạt",
        "columns": {
          "id": "ID",
          "code": "Mã Kích hoạt",
          "template_name": "Tên Mẫu",
          "status": "Trạng thái",
          "expires_at": "Hết hạn",
          "usage_count": "Đã dùng",
          "max_usage": "Khả dụng",
          "created_at": "Ngày tạo"
        }
      },
      "actions": {
        "enable": "Bật",
        "disable": "Tắt",
        "export": "Xuất",
        "exportConfirm": {
          "title": "Xác nhận Xuất",
          "description": "Tất cả mã từ nhóm đã chọn sẽ được xuất ra file văn bản. Tiếp tục?",
          "confirmText": "Xuất"
        }
      },
      "status": {
        "0": "Chưa dùng",
        "1": "Đã dùng",
        "2": "Đã tắt",
        "3": "Hết hạn"
      }
    },
    "usage": {
      "title": "Lịch sử Sử dụng",
      "description": "Xem bản ghi sử dụng thẻ quà tặng.",
      "table": {
        "columns": {
          "id": "ID",
          "code": "Mã Kích hoạt",
          "template_name": "Tên Mẫu",
          "user_email": "Email Người dùng",
          "rewards_given": "Phần thưởng đã cấp",
          "invite_rewards": "Bonus Người mời",
          "multiplier_applied": "Hệ số Áp dụng",
          "ip_address": "Địa chỉ IP",
          "created_at": "Thời gian Sử dụng",
          "actions": "Thao tác"
        }
      },
      "actions": {
        "view": "Chi tiết"
      }
    },
    "statistics": {
      "title": "Thống kê",
      "description": "Phân tích việc sử dụng thẻ quà tặng.",
      "total": {
        "title": "Thống kê Chung",
        "templates_count": "Tổng số Mẫu",
        "active_templates_count": "Mẫu Đang hoạt động",
        "codes_count": "Tổng số Mã Kích hoạt",
        "used_codes_count": "Mã Đã dùng",
        "usages_count": "Bản ghi Kích hoạt"
      },
      "daily": {
        "title": "Sử dụng Trong Ngày"
      }
    }
  },
  "knowledge": {
    "title": "Kho Kiến thức",
    "description": "Tại đây bạn có thể quản lý bài viết kho kiến thức, bao gồm thêm, xóa và sửa.",
    "columns": {
      "id": "ID",
      "status": "Trạng thái",
      "title": "Tiêu đề",
      "category": "Danh mục",
      "actions": "Thao tác"
    },
    "form": {
      "add": "Thêm Bài viết",
      "edit": "Sửa Bài viết",
      "title": "Tiêu đề",
      "titlePlaceholder": "Nhập tiêu đề bài viết",
      "category": "Danh mục",
      "categoryPlaceholder": "Nhập danh mục để tự động nhóm",
      "language": "Ngôn ngữ",
      "languagePlaceholder": "Chọn ngôn ngữ",
      "content": "Nội dung",
      "show": "Hiển thị",
      "cancel": "Hủy",
      "submit": "Gửi"
    },
    "languages": {
      "en-US": "English",
      "ja-JP": "日本語",
      "ko-KR": "한국어",
      "vi-VN": "Tiếng Việt",
      "zh-CN": "简体中文",
      "zh-TW": "繁體中文",
      "ru-RU": "Русский"
    },
    "messages": {
      "deleteConfirm": "Xác nhận Xóa",
      "deleteDescription": "Hành động này sẽ xóa vĩnh viễn bài viết khỏi kho kiến thức. Bạn có chắc chắn muốn tiếp tục?",
      "deleteButton": "Xóa",
      "operationSuccess": "Thao tác thành công"
    },
    "toolbar": {
      "searchPlaceholder": "Tìm trong kho kiến thức...",
      "reset": "Đặt lại",
      "sortModeHint": "Kéo thả để sắp xếp, sau đó nhấn lưu",
      "editSort": "Sửa thứ tự",
      "saveSort": "Lưu thứ tự"
    }
  }
};