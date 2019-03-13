function swSet(sw,val){
   // Feead back provided by data update 
   $.get("set",{'switch':sw,set:val});
}

$(document).ready(function(){
   dataUpdate()
});

function dataUpdate(){
   // update switch status
   // the server blocks for 15 seconds unless data is updated
   $.post('get',{},gotData,'json');
}
function gotData(data){
  $.each( data, function( key, val ) {
    console.log(key+" " + val.Value + " " +val.Status);
    $("#sw-"+key).prop("checked",val.Value).change();
  });
  dataUpdate();// loop
};
