<?php
/**
 * @version    CVS: 1.0.0
 * @package    Com_Subscription
 * @author     Rob <pr@floridarangers.com>
 * @copyright  2024 Rob
 * @license    GNU General Public License version 2 or later; see LICENSE.txt
 */

// No direct access
defined('_JEXEC') or die;

use \Joomla\CMS\HTML\HTMLHelper;
use \Joomla\CMS\Factory;
use \Joomla\CMS\Uri\Uri;
use \Joomla\CMS\Router\Route;
use \Joomla\CMS\Language\Text;
use \Joomla\CMS\Session\Session;
use Joomla\Utilities\ArrayHelper;


?>
<h3 class="componentheading">My Subscription for <?php echo $this->product->name; ?></h3>

<div class="alert alert-primary d-flex align-items-center" role="alert">
  <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="currentColor" class="bi bi-exclamation-triangle-fill flex-shrink-0 me-2" viewBox="0 0 16 16" role="img" aria-label="Warning:">
	<path d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"/>
  </svg>
  <div>
	If you need to update payment methods, cancel this subscription and click on the name to create a new one with your updated payment information
  </div>
</div>
<div class="item_fields">

	<table class="table">
		

		<tr>
			<th><?php echo Text::_('COM_SUBSCRIPTION_FORM_LBL_SUBSCRIBER_USER_ID'); ?></th>
			<td><?php echo $this->item->user_id_name; ?></td>
		</tr>

		<tr>
			<th><?php echo Text::_('COM_SUBSCRIPTION_FORM_LBL_SUBSCRIBER_START_DATE'); ?></th>
			<td>				<?php
			$date = $this->item->start_date;
			echo $date > 0 ? HTMLHelper::_('date', $date, Text::_('DATE_FORMAT_LC4')) : '-';
			?>

			</td>
		</tr>
<tr>
			<th><?php echo Text::_('COM_SUBSCRIPTION_FORM_LBL_SUBSCRIBER_PRODUCT_ID'); ?></th>
			<td>$<?php echo number_format($this->product->cost,2); ?>/<?php echo $this->product->period; ?></td>
		</tr>
		<tr>
			<th><?php echo Text::_('COM_SUBSCRIPTION_FORM_LBL_SUBSCRIBER_PRODUCT_ID'); ?></th>
			<td><?php echo $this->product->name; ?></td>
		</tr>

	<tr>
			<th><?php echo Text::_('COM_SUBSCRIPTION_FORM_LBL_PRODUCT_DESCRIPTION'); ?></th>
			<td><?php echo $this->product->description; ?></td>
		</tr>		
		<?php if ($this->item->state==1) { ?>
		<tr>
			<th>Action</th>
			<td>
				
				<form onsubmit="return confirm('Are you sure you want to cancel the subscription?');"  method="post" action="/index.php?option=com_subscription&task=subscriber.cancel">
					<input type="hidden" name="subscription_id" value="<?php echo $this->item->id; ?>"/>
					<button  class="btn btn-danger" >Cancel Subscription</button>
				</form>
			</td>
		</tr>	
		<?php  }else{  ?>
			<tr>
				<th><?php echo Text::_('COM_SUBSCRIPTION_FORM_LBL_SUBSCRIBER_END_DATE'); ?></th>
				<td>				<?php
				$date = $this->item->end_date;
				echo $date > 0 ? HTMLHelper::_('date', $date, Text::_('DATE_FORMAT_LC4')) : '-';
				?>
			
				</td>
			</tr>
	    <?php } ?>

	</table>

</div>

