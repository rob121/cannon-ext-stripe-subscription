<?php
use \Joomla\CMS\Router\Route;
use \Joomla\CMS\Uri\Uri;
use \Joomla\CMS\HTML\HTMLHelper;

$uri = Uri::getInstance();
HTMLHelper::_('behavior.core');
HTMLHelper::_('behavior.keepalive');

?>

<h3>Checkout: <?php echo $this->item->name; ?></h3>

<div class="card">
<div class="card-header" >
<div class="card-body" >
    <?php if($this->item->test_mode==1){ ?>
    <div class="row mb-3">
        <div class="col">
            <div class="border border-1 border-danger bg-light p-3 border-rounded">
    
       
        <h4 class="text-danger"><i class="fa fa-exclamation" > </i> Test Mode is Enabled, if you see this and are not expecting it, you will not be able to register</h4>
        
        </div>
        </div>
    </div>
    <?php } ?>
    <form id="payment-form">
    <div class="row">
    <div class="col">
    <div id="link-authentication-element">
      <!-- Elements will create authentication element here -->
    </div>
    <div id="payment-element">
      <!-- Elements will create form elements here -->
    </div>
    <div id="error-message">
      <!-- Display error message to your customers here -->
    </div>
    <div class="d-flex justify-content-lg-end justify-content-sm-center" >
    <button id="pay_button" name="save" class="mt-3 btn btn-primary"><span style="display:none"; id="spinner" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>Subscribe Now</button>
    </div>
    </div>
    </div>
    </form>
</div>
</div>
</div>


<script>
    
    let elements;

    jQuery("#payment-form").on("submit",handleSubmit);
    
    var clientSecret = "<?php echo $this->data->client_secret; ?>";
    
    var rurl = "<?php echo Route::_($uri->getScheme()."://".$uri->getHost()."/index.php?option=com_subscription&task=checkout.complete&subscription_id=".$this->data->subscription_id."&customer_id=".$this->data->customer_id."&product_id=".$this->item->id,true); ?>";
    
    const formatter = new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    
      // These options are needed to round to whole numbers if that's what you want.
      //minimumFractionDigits: 0, // (this suffices for whole numbers, but will print 2500.10 as $2,500.1)
      //maximumFractionDigits: 0, // (causes 2500.99 to be printed as $2,501)
    });
    

	jQuery(function(){ 
	   
		
				     elements = stripe.elements({clientSecret: clientSecret });
				   
					 const paymentElement = elements.create('payment');
					 paymentElement.mount('#payment-element');
					 // Create and mount the linkAuthentication Element to enable autofilling customer payment details
					// const linkAuthenticationElement = elements.create("linkAuthentication");
					// linkAuthenticationElement.mount("#link-authentication-element");
				   

	 }); 
		


async function handleSubmit(e) {
  e.preventDefault();

  // Disable button + show spinner (note: "disabled", not "disable")
  jQuery("#pay_button").prop("disabled", true);
  jQuery("#spinner").show();

  // ✅ MUST be awaited, and done before confirmPayment
  const { error: submitError } = await elements.submit();
  if (submitError) {
    showMessage(submitError.message);
    jQuery("#pay_button").prop("disabled", false);
    jQuery("#spinner").hide();
    return;
  }

  const { error } = await stripe.confirmPayment({
    elements,
    clientSecret,
    confirmParams: {
      return_url: rurl,
    },
  });

  // Only reached on immediate error (otherwise redirect happens)
  if (error) {
    if (error.type === "card_error" || error.type === "validation_error") {
      showMessage(error.message);
    } else {
      showMessage("An unexpected error occurred.");
    }
    jQuery("#pay_button").prop("disabled", false);
    jQuery("#spinner").hide();
  }
}
	
	
	function showMessage(messageText) {
		jQuery("#spinner").hide(); 
	  const messageContainer = document.querySelector("#payment-message");
	
	  messageContainer.classList.remove("hidden");
	  messageContainer.textContent = messageText;
	
	  setTimeout(function () {
		messageContainer.classList.add("hidden");
		messageContainer.textContent = "";
	  }, 4000);
	}     

</script>
