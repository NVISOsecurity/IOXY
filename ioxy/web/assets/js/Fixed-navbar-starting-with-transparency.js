$(document).ready( function() {
    function checkScroll(){
        var opacity = 150; //start point navbar fixed to top changes in px

        if($(window).scrollTop() > opacity){
            $('.navbar.navbar-fixed-top').addClass("navchange");
        }else{
            $('.navbar.navbar-fixed-top').removeClass("navchange");
        }
    }

    if($('.navbar').length > 0){
        $(window).on("scroll load resize", function(){
            checkScroll();
        });
    }
    
    $('.dropdown').on('show.bs.dropdown', function() {
        $(this).find('.dropdown-menu').first().stop(true, true).slideDown(300);
    });

    $('.dropdown').on('hide.bs.dropdown', function() {
        $(this).find('.dropdown-menu').first().stop(true, true).slideUp(300);
    });
    
})