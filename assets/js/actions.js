function initActions() {
    // error message modal
    var errorModal = new bootstrap.Modal($('#error-modal'), { keyboard: true });
    // success message modal
    var successModal = new bootstrap.Modal($('#success-modal'), { keyboard: true });

    // enabled
    $(".action-enable-strategy").click(function(e) {
        // prevent link from scrolling up
        e.preventDefault();

        uuid = $(this).data("uuid");
        $.ajax({
            type: 'GET',
            url: '/action/enable_strategy/' + uuid,
            data: {},
            success: function() {
                $('#success-modal-body').text("啟動中, 請留意通知, 即將重整頁面");
                successModal.show();
                setTimeout(function(){
                    window.location.reload(1);
                }, 400);
            },
        }).fail(function(data) {
            $('#error-modal-body').text(data.responseJSON.error);
            errorModal.show();
        });
    });

    // disable
    $(".action-disable-strategy").click(function(e) {
        // prevent link from scrolling up
        e.preventDefault();

        uuid = $(this).data("uuid");
        $.ajax({
            type: 'GET',
            url: '/action/disable_strategy/' + uuid,
            data: {},
            success: function() {
                $('#success-modal-body').text("關閉中, 請留意通知, 即將重整頁面");
                successModal.show();
                setTimeout(function(){
                    window.location.reload(1);
                }, 400);
            },
        }).fail(function(data) {
            $('#error-modal-body').text(data.responseJSON.error);
            errorModal.show();
        });
    });

    // reset
    $(".action-reset-strategy").click(function(e) {
        // prevent link from scrolling up
        e.preventDefault();

        uuid = $(this).data("uuid");
        if (!confirm("確定要重置狀態嗎?")) {
            // Close dropdown menu
            $('#actions-dropdown-'+uuid).dropdown('toggle');
            return false;
        }

        $.ajax({
            type: 'GET',
            url: '/action/reset_strategy/' + uuid,
            data: {},
            success: function() {
                $('#success-modal-body').text("已成功重置狀態, 即將重整頁面");
                successModal.show();
                setTimeout(function(){
                    window.location.reload(1);
                }, 400);
            },
        }).fail(function(data) {
            $('#error-modal-body').text(data.responseJSON.error);
            errorModal.show();
        });
    });

    // delete
    $(".action-delete-strategy").click(function(e) {
        // prevent link from scrolling up
        e.preventDefault();

        uuid = $(this).data("uuid");
        if (!confirm("確定要刪除嗎?")) {
            // Close dropdown menu
            $('#actions-dropdown-'+uuid).dropdown('toggle');
            return false;
        }

        $.ajax({
            type: 'DELETE',
            url: '/strategy/' + uuid,
            data: {},
            success: function() {
                $('#success-modal-body').text("已成功刪除, 即將重整頁面");
                successModal.show();
                setTimeout(function(){
                    window.location.reload(1);
                }, 400);
            },
        }).fail(function(data) {
            $('#error-modal-body').text(data.responseJSON.error);
            errorModal.show();
        });
    });

    // close position
    $(".action-close-position").click(function(e) {
        // prevent link from scrolling up
        e.preventDefault();

        uuid = $(this).data("uuid");
        if (!confirm("確定要平倉嗎?")) {
            // Close dropdown menu
            $('#actions-dropdown-'+uuid).dropdown('toggle');
            return false;
        }

        $.ajax({
            type: 'GET',
            url: '/action/close_position/' + uuid,
            data: {},
            success: function(data) {
                msg = "已成功平倉, 即將重整頁面";
                $('#success-modal-body').text(msg);
                successModal.show();
                setTimeout(function(){
                    window.location.reload(1);
                }, 400);
            },
        }).fail(function(data) {
            $('#error-modal-body').text(data.responseJSON.error);
            errorModal.show();
        });
    });
}
