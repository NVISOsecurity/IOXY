$(document).ready(function(){
    $("#alert").fadeTo(2000, 500).slideUp(500, function() {
      $("#alert").slideUp(500);
    });
    $("#options-eb").change(function(){
        $(".content-eb").addClass("d-none");
        $("#content-eb-"+$(this).val()).removeClass("d-none");
        $(".content-eb :input").attr("disabled", true);
        $("#content-eb-"+$(this).val()+" :input").attr("disabled", false);
    });
    $("#options-b").change(function(){
        $(".content-b").addClass("d-none");
        $("#content-b-"+$(this).val()).removeClass("d-none");
        $(".content-b :input").attr("disabled", true);
        $("#content-b-"+$(this).val()+" :input").attr("disabled", false);
    });
    $("#startProxy").click(function(){
        $.ajax({
            url: '/settings',
            type: 'POST',
            data: 'start=true',
            dataType: 'html',
            success: function (data, textStatus, XmlHttpRequest) {
                if (XmlHttpRequest.status === 200) {
                    $("#Status").text($(XmlHttpRequest.responseText).find('#Status').text());
                    setTimeout(function() {$.ajax({
                        url: "/settings",
                        type: 'GET',
                        success: function (data, textStatus, XmlHttpRequest) {
                            if (XmlHttpRequest.status === 200) {
                                $("#Status").text($(XmlHttpRequest.responseText).find('#Status').text());
                                $("#applink").removeClass("disableDiv");
                                $("#settingsform").addClass("disableDiv");
                            }
                        }
                    })}, 2000);
                }
            }
        });
    });
    $("#stopProxy").click(function(){
        $.ajax({
           url : '/settings',
           type : 'POST',
           data : 'start=false',
           dataType : 'html',
           success: function(data, textStatus, XmlHttpRequest) {
             if (XmlHttpRequest.status === 200) {  
                 $("#Status").text($(XmlHttpRequest.responseText).find('#Status').text());
                 $("#settingsform").removeClass("disableDiv");
                 $("#applink").addClass("disableDiv");
             }
            }
        });
    });
});