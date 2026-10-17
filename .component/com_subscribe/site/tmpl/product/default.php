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

$redirectUrl = Route::_('index.php?option=com_subscription&view=product&id='.$this->item->id);
$ret =  base64_encode($redirectUrl);
?>

<h3>	<?php echo nl2br($this->item->name); ?></h3>
<div class="card">
	<div class="card-title">
		
	
	</div>
	<div class="card-body">
		
		<?php echo nl2br($this->item->description); ?>
		
	</div>
	<div class="card-footer">

		<?php if ($this->user->get('id')>0){ ?>
				
				<a href="<?php echo Route::_("index.php?option=com_subscription&view=checkout&id=".$this->item->id); ?>" class="btn btn-primary" >Subscribe for $<?php echo number_format($this->item->cost,2); ?>/<?php echo Text::_('COM_SUBSCRIPTION_PRODUCTS_PERIOD_OPTION_' . preg_replace('/[^A-Za-z0-9\_-]/', '',strtoupper(str_replace(' ', '_',$this->item->period)))); ?></a>
					
		<?php }else{ ?>  
			
			    <a href="<?php echo Route::_('index.php?option=com_users&view=login&return='.$ret); ?>" class="btn btn-danger" >You must login before subscribing!</a>
			
		<?php } ?>
	</div>

</div>